package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ghostlawless/xdl/internal/config"
	"github.com/ghostlawless/xdl/internal/httpx"
)

type UserTweetsVariables struct {
	UserID                            string `json:"userId"`
	Count                             int    `json:"count"`
	IncludePromotedContent            bool   `json:"includePromotedContent"`
	WithQuickPromoteEligibilityFields bool   `json:"withQuickPromoteEligibilityTweetFields"`
	WithVoice                         bool   `json:"withVoice"`
}

func BuildUserTweetsParams(userID string, count int) (url.Values, error) {
	if count <= 0 {
		count = 20
	}

	vars := UserTweetsVariables{
		UserID:                            userID,
		Count:                             count,
		IncludePromotedContent:            true,
		WithQuickPromoteEligibilityFields: true,
		WithVoice:                         true,
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, fmt.Errorf("encode variables: %w", err)
	}

	features := map[string]bool{
		"rweb_video_screen_enabled":                                               false,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"responsive_web_profile_redirect_enabled":                                 false,
		"rweb_tipjar_consumption_enabled":                                         false,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      true,
		"responsive_web_jetfuel_frame":                                            true,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_grok_imagine_annotation_enabled":                          true,
		"responsive_web_grok_community_note_auto_translation_is_enabled":          false,
		"responsive_web_enhance_cards_enabled":                                    false,
	}

	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("encode features: %w", err)
	}

	fieldToggles := map[string]bool{
		"withArticlePlainText": false,
	}

	fieldTogglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, fmt.Errorf("encode fieldToggles: %w", err)
	}

	params := url.Values{}
	params.Set("variables", string(varsJSON))
	params.Set("features", string(featuresJSON))
	params.Set("fieldToggles", string(fieldTogglesJSON))

	return params, nil
}

func FetchUserTweetsPage(
	ctx context.Context,
	client *http.Client,
	conf *config.EssentialsConfig,
	userID string,
	count int,
) (*httpx.Response, error) {
	if client == nil || conf == nil {
		return nil, fmt.Errorf("nil client or config")
	}
	if userID == "" {
		return nil, fmt.Errorf("empty userID")
	}

	endpoint, err := conf.GraphQLURL("user_tweets")
	if err != nil {
		return nil, err
	}

	params, err := BuildUserTweetsParams(userID, count)
	if err != nil {
		return nil, err
	}

	opt := httpx.RequestOptionsRuntime{
		Method:      http.MethodGet,
		URI:         endpoint,
		Params:      params,
		Headers:     http.Header{},
		Timeout:     15 * time.Second,
		WithCookies: true,
	}

	resp, err := httpx.DoRequest(ctx, client, opt)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UserTweets HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	return resp, nil

}
