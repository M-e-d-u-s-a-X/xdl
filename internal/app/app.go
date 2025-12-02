package app

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ghostlawless/xdl/internal/config"
	"github.com/ghostlawless/xdl/internal/downloader"
	"github.com/ghostlawless/xdl/internal/log"
	xruntime "github.com/ghostlawless/xdl/internal/runtime"
	"github.com/ghostlawless/xdl/internal/scraper"
	"github.com/ghostlawless/xdl/internal/utils"
)

type RunMode int

const (
	ModeVerbose RunMode = iota
	ModeQuiet
	ModeDebug
)

type RunContext struct {
	Users             []string
	Mode              RunMode
	RunID             string
	RunSeed           []byte
	LogPath           string
	CookiePath        string
	CookiePersistPath string
	OutRoot           string
	NoDownload        bool
	DryRun            bool
}

var termMu sync.Mutex

type interactiveControl struct {
	mu     sync.RWMutex
	paused bool
	quit   bool
}

func (c *interactiveControl) ShouldPause() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paused
}

func (c *interactiveControl) ShouldQuit() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.quit
}

func (c *interactiveControl) setPaused(v bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.paused = v
	c.mu.Unlock()
}

func (c *interactiveControl) setQuit() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.quit = true
	c.paused = false
	c.mu.Unlock()
}

var globalControl = &interactiveControl{}

func startKeyboardControlListener(c *interactiveControl) {
	if c == nil {
		return
	}
	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			ch, err := r.ReadByte()
			if err != nil {
				return
			}
			switch ch {
			case 'p', 'P':
				c.setPaused(true)
				termMu.Lock()
				fmt.Print("\r\033[2K\033[33;1mxdl ▸ paused. press 'c' to continue or 'q' to quit.\033[0m\n")
				termMu.Unlock()
				for {
					ch2, err2 := r.ReadByte()
					if err2 != nil {
						return
					}
					switch ch2 {
					case 'c', 'C':
						c.setPaused(false)
						termMu.Lock()
						fmt.Print("\033[32;1mxdl ▸ resuming...\033[0m\n")
						termMu.Unlock()
						goto nextKey
					case 'q', 'Q':
						c.setQuit()
						termMu.Lock()
						fmt.Print("\r\033[2K\033[31;1mxdl ▸ quit requested. finishing current cycle...\033[0m\n")
						termMu.Unlock()
						return
					}
				}
			case 'q', 'Q':
				c.setQuit()
				termMu.Lock()
				fmt.Print("\r\033[2K\033[31;1mxdl ▸ quit requested. finishing current cycle...\033[0m\n")
				termMu.Unlock()
				return
			}
		nextKey:
		}
	}()
}

type spinner struct {
	prefix string
	stop   chan struct{}
	done   chan struct{}
}

func startSpinner(prefix string) *spinner {
	s := &spinner{prefix: prefix, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(s.done)
		frames := []rune{'|', '/', '-', '\\'}
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
			}
			termMu.Lock()
			fmt.Printf("\r%s [%c]", s.prefix, frames[i%len(frames)])
			termMu.Unlock()
			i++
			time.Sleep(120 * time.Millisecond)
		}
	}()
	return s
}

func (s *spinner) Stop() {
	if s == nil {
		return
	}
	close(s.stop)
	<-s.done
	termMu.Lock()
	fmt.Print("\r")
	termMu.Unlock()
}

func generateRunID() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func buildAPIClient(t time.Duration) *http.Client {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	if t <= 0 {
		t = 15 * time.Second
	}
	return &http.Client{Transport: tr, Timeout: t}
}

func buildDownloadClient() *http.Client {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   7 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &http.Client{Transport: tr, Timeout: 0}
}

func parseArgs(args []string, presetRunID string, presetRunSeed []byte) (RunContext, error) {
	var (
		fQuiet             bool
		fDebug             bool
		fCookiePath        string
		fCookiePersistPath string
	)
	for _, a := range args {
		switch a {
		case "-q", "/q":
			fQuiet = true
		case "-d", "/d":
			fDebug = true
		}
	}
	fs := flag.NewFlagSet("xdl", flag.ContinueOnError)
	fs.BoolVar(&fQuiet, "q", fQuiet, "Quiet mode")
	fs.BoolVar(&fDebug, "d", fDebug, "Debug mode")
	fs.StringVar(&fCookiePath, "c", "", "Cookie JSON file exported from browser extension")
	fs.StringVar(&fCookiePersistPath, "C", "", "Cookie JSON file to import and persist into essentials.json")
	if err := fs.Parse(args); err != nil {
		return RunContext{}, err
	}
	rest := fs.Args()
	if (len(rest) == 0 || rest[0] == "") && fCookiePersistPath != "" {
		ctx := RunContext{
			Users:             nil,
			Mode:              ModeVerbose,
			RunID:             presetRunID,
			RunSeed:           presetRunSeed,
			CookiePath:        "",
			CookiePersistPath: fCookiePersistPath,
			OutRoot:           "xDownloads",
			NoDownload:        true,
			DryRun:            false,
		}
		if fDebug {
			ctx.Mode = ModeDebug
		} else if fQuiet {
			ctx.Mode = ModeQuiet
		}
		if ctx.RunID == "" {
			ctx.RunID = generateRunID()
		}
		if ctx.Mode == ModeDebug {
			ctx.LogPath = filepath.Join("logs", "run_"+ctx.RunID)
			if err := os.MkdirAll(ctx.LogPath, 0o755); err != nil {
				return RunContext{}, fmt.Errorf("failed to create log dir: %w", err)
			}
			log.Init(filepath.Join(ctx.LogPath, "main.log"))
			log.LogInfo("main", "Debug mode enabled; logs stored in "+ctx.LogPath)
		} else {
			log.Disable()
		}
		return ctx, nil
	}
	if len(rest) == 0 || rest[0] == "" {
		return RunContext{}, fmt.Errorf("usage: xdl [-q|-d] [-c cookies.json] <username> [more_usernames...] or xdl -C cookies.json (defaults to config/cookies.json when -c is omitted)")
	}
	users := make([]string, 0, len(rest))
	for _, u := range rest {
		if u == "" {
			continue
		}
		if u == "-d" || u == "/d" || u == "-q" || u == "/q" {
			continue
		}
		users = append(users, u)
	}
	if len(users) == 0 {
		return RunContext{}, fmt.Errorf("usage: xdl [-q|-d] [-c cookies.json] <username> [more_usernames...] or xdl -C cookies.json (defaults to config/cookies.json when -c is omitted)")
	}
	ctx := RunContext{
		Users:             users,
		Mode:              ModeVerbose,
		RunID:             presetRunID,
		RunSeed:           presetRunSeed,
		CookiePath:        fCookiePath,
		CookiePersistPath: fCookiePersistPath,
		OutRoot:           "xDownloads",
		NoDownload:        false,
		DryRun:            false,
	}
	if fDebug {
		ctx.Mode = ModeDebug
	} else if fQuiet {
		ctx.Mode = ModeQuiet
	}
	if ctx.RunID == "" {
		ctx.RunID = generateRunID()
	}
	if ctx.Mode == ModeDebug {
		ctx.LogPath = filepath.Join("logs", "run_"+ctx.RunID)
		if err := os.MkdirAll(ctx.LogPath, 0o755); err != nil {
			return RunContext{}, fmt.Errorf("failed to create log dir: %w", err)
		}
		log.Init(filepath.Join(ctx.LogPath, "main.log"))
		log.LogInfo("main", "Debug mode enabled; logs stored in "+ctx.LogPath)
	} else {
		log.Disable()
	}
	return ctx, nil
}

func RunWithArgsAndID(args []string, runID string, runSeed []byte) error {
	rctx, err := parseArgs(args, runID, runSeed)
	if err != nil {
		return err
	}
	return runWithContext(rctx)
}

func RunWithArgs(args []string) error {
	return RunWithArgsAndID(args, "", nil)
}

func Run() {
	if err := RunWithArgsAndID(os.Args[1:], "", nil); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
	}
}

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
	if rctx.CookiePersistPath != "" {
		if essentialsPath == "" {
			if rctx.Mode == ModeVerbose {
				utils.PrintError("cannot persist cookies: essentials.json not found on disk")
			}
			log.LogError("config", "cannot persist cookies: essentials.json not found on disk")
			return fmt.Errorf("essentials.json not found on disk")
		}
		if err := config.ApplyCookiesFromFileAndPersist(conf, rctx.CookiePersistPath, essentialsPath); err != nil {
			if rctx.Mode == ModeVerbose {
				utils.PrintError("failed to apply and persist cookies: %v", err)
			}
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

	if rctx.CookiePath == "" {
		defaultCookiePath := filepath.Join("config", "cookies.json")
		if _, err := os.Stat(defaultCookiePath); err == nil {
			rctx.CookiePath = defaultCookiePath
		}
	}

	if rctx.CookiePath != "" {
		if err := config.ApplyCookiesFromFile(conf, rctx.CookiePath); err != nil {
			if rctx.Mode == ModeVerbose {
				utils.PrintError("failed to apply cookies: %v", err)
			}
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
	lim := xruntime.NewLimiterWith(rctx.RunSeed, []byte(strings.TrimSpace(conf.Runtime.LimiterSecret)))
	if rctx.Mode == ModeDebug {
		log.LogInfo("main", fmt.Sprintf("xdl start | run_id=%s | target=%s", rctx.RunID, username))
	}
	var spin *spinner
	if rctx.Mode == ModeVerbose {
		label := fmt.Sprintf("scanning media for target @%s", username)
		if len(rctx.Users) == 1 {
			utils.PrintInfo("%s", label)
		} else {
			spin = startSpinner(label)
		}
	}
	vb := rctx.Mode == ModeVerbose && len(rctx.Users) == 1
	uid, err := scraper.FetchUserID(apiClient, conf, username)
	if err != nil {
		if spin != nil {
			spin.Stop()
		}
		if rctx.Mode == ModeVerbose {
			utils.PrintError("user lookup failed for @%s: %v", username, err)
		}
		log.LogError("user", err.Error())
		return err
	}
	if rctx.Mode == ModeDebug {
		log.LogInfo("user", "["+uid+"]")
	}
	links, err := scraper.GetMediaLinksForUser(apiClient, conf, uid, username, vb, lim)
	if spin != nil {
		spin.Stop()
	}
	if err != nil && rctx.Mode == ModeVerbose {
		utils.PrintWarn("media listing error for @%s: %v", username, err)
	}
	total := len(links)
	ph := 0
	vd := 0
	for _, m := range links {
		if m.Type == "image" {
			ph++
		} else if m.Type == "video" {
			vd++
		}
	}
	if rctx.Mode == ModeDebug {
		log.LogInfo("media", fmt.Sprintf("media found: %d", total))
	} else if rctx.Mode == ModeVerbose {
		utils.PrintSuccess("timeline scanned - media: %d [photo %d, video %d]", total, ph, vd)
	}
	runDirName := username
	runDir := filepath.Join(rctx.OutRoot, runDirName)
	if err := utils.EnsureDir(rctx.OutRoot); err != nil {
		return err
	}
	if utils.DirExists(runDir) {
		i := 1
		for {
			cn := fmt.Sprintf("%s_%03d", username, i)
			cp := filepath.Join(rctx.OutRoot, cn)
			if !utils.DirExists(cp) {
				runDirName = cn
				runDir = cp
				break
			}
			i++
			if i > 9999 {
				return fmt.Errorf("failed to allocate output folder for @%s", username)
			}
		}
	} else {
		if err := utils.EnsureDir(runDir); err != nil {
			return err
		}
	}
	if !rctx.NoDownload {
		var okc, skc, flc int
		var bytes int64
		var evc int
		var cb func(ev downloader.ProgressEvent)
		if total > 0 {
			switch rctx.Mode {
			case ModeVerbose:
				cb = func(ev downloader.ProgressEvent) {
					if globalControl.ShouldQuit() {
						return
					}
					termMu.Lock()
					defer termMu.Unlock()
					switch ev.Kind {
					case downloader.ProgressKindDownloaded:
						okc++
						bytes += ev.Size
					case downloader.ProgressKindSkipped:
						skc++
					case downloader.ProgressKindFailed:
						flc++
					}
					done := okc + skc + flc
					if total <= 0 {
						return
					}
					f := float64(done) / float64(total)
					if f < 0 {
						f = 0
					}
					if f > 1 {
						f = 1
					}
					p := f * 100.0
					bar := buildProgressBar(30, f)
					sfx := ""
					if globalControl.ShouldPause() {
						sfx = " [paused]"
					}
					fmt.Printf("\r\033[36;1mxdl ▸ [@%s]%s [%s] %3.0f%% %d/%d (ok:%d skip:%d fail:%d)\033[0m",
						username, sfx, bar, p, done, total, okc, skc, flc)
				}
			case ModeDebug:
				cb = func(ev downloader.ProgressEvent) {
					switch ev.Kind {
					case downloader.ProgressKindDownloaded:
						okc++
						bytes += ev.Size
					case downloader.ProgressKindSkipped:
						skc++
					case downloader.ProgressKindFailed:
						flc++
					}
					done := okc + skc + flc
					if total <= 0 {
						return
					}
					evc++
					logNow := false
					if total <= 50 {
						logNow = true
					} else if evc%10 == 0 || done == total {
						logNow = true
					}
					if !logNow {
						return
					}
					f := float64(done) / float64(total)
					if f < 0 {
						f = 0
					}
					if f > 1 {
						f = 1
					}
					percent := int(f*100 + 0.5)
					const (
						cC  = "\033[36m"
						cG  = "\033[32m"
						cY  = "\033[33m"
						cR  = "\033[31m"
						cD  = "\033[2m"
						cRS = "\033[0m"
					)
					sc := cG
					if flc > 0 {
						sc = cR
					} else if skc > 0 {
						sc = cY
					}
					msg := fmt.Sprintf("%sprogress%s user=%s%s%s done=%s%d/%d%s (%s%d%%%s) ok=%s%d%s skip=%s%d%s fail=%s%d%s bytes=%s%d%s",
						cC, cRS, cD, username, cRS, sc, done, total, cRS, cC, percent, cRS, cG, okc, cRS, cY, skc, cRS, cR, flc, cRS, cD, bytes, cRS)
					log.LogInfo("download", msg)
				}
			}
		}
		if rctx.Mode == ModeVerbose {
			utils.PrintInfo("output: %s", runDir)
		}
		summary, derr := downloader.DownloadAllCycles(dlClient, conf, links, downloader.Options{
			RunDir:            runDir,
			User:              username,
			MediaMaxBytes:     0,
			DryRun:            rctx.DryRun,
			Attempts:          3,
			PerAttemptTimeout: 2 * time.Minute,
			Progress:          cb,
			ShouldPause:       globalControl.ShouldPause,
			ShouldQuit:        globalControl.ShouldQuit,
		})
		if rctx.Mode == ModeVerbose && total > 0 && cb != nil {
			termMu.Lock()
			fmt.Print("\n")
			termMu.Unlock()
		}
		if rctx.Mode == ModeDebug {
			log.LogInfo("download", fmt.Sprintf("done: ok=%d skipped=%d failed=%d bytes=%d cycles=%d",
				summary.Downloaded, summary.Skipped, summary.Failed, summary.TotalBytes, summary.Cycles))
			log.LogInfo("main", fmt.Sprintf("xdl[%s] exit [%.2fs] user=%s", rctx.RunID, time.Since(start).Seconds(), username))
		} else if rctx.Mode == ModeVerbose {
			totalMB := float64(summary.TotalBytes) / 1024.0 / 1024.0
			utils.PrintSuccess("complete @%s — ok:%d skip:%d fail:%d (%.2f MB, %.2fs)",
				username, summary.Downloaded, summary.Skipped, summary.Failed, totalMB, time.Since(start).Seconds())
		}
		if derr != nil {
			log.LogError("download", derr.Error())
		}
		if globalControl.ShouldQuit() {
			if rctx.Mode == ModeVerbose {
				utils.PrintWarn("run aborted by user for @%s", username)
			}
			return fmt.Errorf("aborted by user")
		}
	}
	return nil
}

func buildProgressBar(width int, fraction float64) string {
	if width <= 0 {
		width = 20
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(float64(width)*fraction + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	b := make([]byte, width)
	for i := 0; i < width; i++ {
		if i < filled {
			b[i] = '='
		} else {
			b[i] = ' '
		}
	}
	return string(b)
}
