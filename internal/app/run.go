package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ghostlawless/xdl/internal/config"
	"github.com/ghostlawless/xdl/internal/log"
	"github.com/ghostlawless/xdl/internal/runtime"
	"github.com/ghostlawless/xdl/internal/utils"
)

func runWithContext(rctx RunContext) error {
	_ = context.Background()

	if rctx.Mode == ModeVerbose {
		utils.PrintBanner()
	}

	startKeyboardControlListener(globalControl)

	essentialsCandidates := []string{
		filepath.Join(".", "config", "essentials.json"),
		filepath.Join(".", "essentials.json"),
	}

	conf, err := config.LoadEssentialsWithFallback(essentialsCandidates)
	if err != nil {
		if rctx.Mode == ModeVerbose {
			utils.PrintError("failed to load essentials: %v", err)
		}
		log.LogError("config", "failed to load essentials: "+err.Error())
		return err
	}

	if rctx.Mode == ModeDebug {
		conf.Paths.Debug = rctx.LogPath
		conf.Paths.DebugRaw = rctx.LogPath
	}

	essentialsPath := ""
	for _, p := range essentialsCandidates {
		if _, err := os.Stat(p); err == nil {
			essentialsPath = p
			break
		}
	}

	// Persist cookies (optional feature).
	if rctx.CookiePersistPath != "" {
		if essentialsPath == "" {
			utils.PrintError("MISSING essentials.json on disk: cannot persist cookies")
			log.LogError("config", "cannot persist cookies: essentials.json not found on disk")
			return fmt.Errorf("essentials.json not found on disk")
		}

		if err := config.ApplyCookiesFromFileAndPersist(conf, rctx.CookiePersistPath, essentialsPath); err != nil {
			// Always show cookie-related errors to the user.
			utils.PrintError("%v", err)
			log.LogError("config", "failed to apply and persist cookies: "+err.Error())
			return err
		}

		if rctx.Mode == ModeDebug {
			hasGuest := conf.Auth.Cookies.GuestID != ""
			hasAuth := conf.Auth.Cookies.AuthToken != ""
			hasCt0 := conf.Auth.Cookies.Ct0 != ""
			log.LogInfo("config", fmt.Sprintf("cookies persisted: guest_id=%v auth_token=%v ct0=%v into %s", hasGuest, hasAuth, hasCt0, essentialsPath))
		} else if rctx.Mode == ModeVerbose {
			utils.PrintSuccess("cookies imported and persisted into %s", essentialsPath)
		}

		if len(rctx.Users) == 0 {
			return nil
		}
	}

	// Auto-detect default cookies.json.
	defaultCookiePath := filepath.Join("config", "cookies.json")
	if rctx.CookiePath == "" {
		if _, err := os.Stat(defaultCookiePath); err == nil {
			rctx.CookiePath = defaultCookiePath
		}
	}

	// Load cookies from file (if provided/detected).
	if rctx.CookiePath != "" {
		if err := config.ApplyCookiesFromFile(conf, rctx.CookiePath); err != nil {
			// Always show cookie-related errors to the user.
			utils.PrintError("%v", err)
			log.LogError("config", "failed to apply cookies: "+err.Error())
			return err
		}

		if rctx.Mode == ModeDebug {
			hasGuest := conf.Auth.Cookies.GuestID != ""
			hasAuth := conf.Auth.Cookies.AuthToken != ""
			hasCt0 := conf.Auth.Cookies.Ct0 != ""
			log.LogInfo("config", fmt.Sprintf("cookies loaded: guest_id=%v auth_token=%v ct0=%v", hasGuest, hasAuth, hasCt0))
		}
	}

	// HARD REQUIREMENT: fail fast if cookies are missing.
	cookieHintPath := rctx.CookiePath
	if cookieHintPath == "" {
		cookieHintPath = defaultCookiePath
	}

	missing := make([]string, 0, 2)
	if strings.TrimSpace(conf.Auth.Cookies.AuthToken) == "" {
		missing = append(missing, "auth_token")
	}
	if strings.TrimSpace(conf.Auth.Cookies.Ct0) == "" {
		missing = append(missing, "ct0")
	}
	if len(missing) > 0 {
		e := fmt.Errorf(
			"MISSING COOKIES: %s.\nFix: login to x.com, export cookies as JSON (Cookie-Editor), save to %q, then run again.",
			strings.Join(missing, ", "),
			cookieHintPath,
		)
		utils.PrintError("%v", e)
		log.LogError("config", e.Error())
		return e
	}

	apiTimeout := conf.HTTPTimeout()
	apiClient := buildAPIClient(apiTimeout)
	dlClient := buildDownloadClient()

	if len(rctx.Users) == 1 {
		return runSingleUser(rctx, conf, apiClient, dlClient, rctx.Users[0])
	}

	cc := len(rctx.Users)
	if cc > 4 {
		cc = 4
	}

	errCh := make(chan error, len(rctx.Users))
	sem := make(chan struct{}, cc)

	var wg sync.WaitGroup
	for _, user := range rctx.Users {
		u := user
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := runSingleUser(rctx, conf, apiClient, dlClient, u); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return e
		}
	}

	return nil

}

func runSingleUser(rctx RunContext, conf *config.EssentialsConfig, apiClient, dlClient *http.Client, username string) error {
	start := time.Now()
	lim := runtime.NewLimiterWith(rctx.RunSeed, []byte(strings.TrimSpace(conf.Runtime.LimiterSecret)))

	if rctx.Mode == ModeDebug {
		log.LogInfo("main", fmt.Sprintf("xdl start | run_id=%s | target=%s", rctx.RunID, username))
	}
	if rctx.Mode == ModeVerbose {
		utils.PrintInfo("loading profile target [@%s]...", username)
	}

	spin := newSpinnerForUser(rctx, username)
	if spin != nil {
		defer stopSpinner(spin)
	}

	runDir, err := prepareRunOutputDir(rctx, conf, username, spin)
	if err != nil {
		return err
	}

	uid, err := resolveUserID(rctx, conf, apiClient, username, spin)
	if err != nil {
		return err
	}

	scan, stats, err := scanAndDownloadUserMedia(rctx, conf, apiClient, dlClient, uid, username, runDir, lim)
	if err != nil {
		return err
	}

	printRunSummary(rctx, username, start, scan, stats)

	return nil
}
