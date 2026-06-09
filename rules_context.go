package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type PromptRule struct {
	ID        int64  `json:"id"`
	RuleKey   string `json:"rule_key"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Enabled   bool   `json:"enabled"`
	Version   int    `json:"version"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UpdatePromptRuleInput struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}

type PromptResearchSource struct {
	ID               int64  `json:"id"`
	SourceType       string `json:"source_type"`
	TrustTier        string `json:"trust_tier"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	ExtractedPattern string `json:"extracted_pattern"`
	AppAdaptation    string `json:"app_adaptation"`
	AccessedAt       string `json:"accessed_at"`
	CreatedAt        string `json:"created_at"`
}

func (s *Store) seedPromptDefaults(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, rule := range defaultPromptRules() {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO prompt_rules (rule_key, category, title, content, enabled, version, source, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, 1, ?, ?, ?)
			ON CONFLICT(rule_key) DO NOTHING`,
			rule.RuleKey,
			rule.Category,
			rule.Title,
			rule.Content,
			rule.Source,
			now,
			now,
		); err != nil {
			return err
		}
	}
	for _, source := range defaultPromptResearchSources() {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO prompt_research_sources (source_type, trust_tier, title, url, extracted_pattern, app_adaptation, accessed_at, created_at)
			SELECT ?, ?, ?, ?, ?, ?, ?, ?
			WHERE NOT EXISTS (SELECT 1 FROM prompt_research_sources WHERE url = ?)`,
			source.SourceType,
			source.TrustTier,
			source.Title,
			source.URL,
			source.ExtractedPattern,
			source.AppAdaptation,
			source.AccessedAt,
			now,
			source.URL,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListPromptRules() ([]PromptRule, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, rule_key, category, title, content, enabled, version, source, created_at, updated_at
		FROM prompt_rules ORDER BY category, rule_key`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromptRules(rows)
}

func (s *Store) UpdatePromptRule(input UpdatePromptRuleInput) (PromptRule, error) {
	if input.ID <= 0 {
		return PromptRule{}, errors.New("prompt rule id is required")
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return PromptRule{}, errors.New("prompt rule content is required")
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE prompt_rules SET content = ?, enabled = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		content,
		boolToInt(input.Enabled),
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return PromptRule{}, err
	}
	return s.getPromptRule(input.ID)
}

func (s *Store) ListPromptResearchSources() ([]PromptResearchSource, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_type, trust_tier, title, url, extracted_pattern, app_adaptation, accessed_at, created_at
		FROM prompt_research_sources ORDER BY CASE trust_tier WHEN 'official' THEN 0 WHEN 'reputable' THEN 1 WHEN 'prompt_bank' THEN 2 ELSE 3 END, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sources := []PromptResearchSource{}
	for rows.Next() {
		var source PromptResearchSource
		if err := rows.Scan(&source.ID, &source.SourceType, &source.TrustTier, &source.Title, &source.URL, &source.ExtractedPattern, &source.AppAdaptation, &source.AccessedAt, &source.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (s *Store) promptRuleDigest(categories ...string) string {
	categorySet := map[string]bool{}
	for _, category := range categories {
		categorySet[category] = true
	}
	rules, err := s.ListPromptRules()
	if err != nil {
		return ""
	}
	parts := []string{}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if len(categorySet) > 0 && !categorySet[rule.Category] {
			continue
		}
		parts = append(parts, rule.Title+": "+rule.Content)
	}
	return strings.Join(parts, "\n")
}

func (s *Store) getPromptRule(id int64) (PromptRule, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, rule_key, category, title, content, enabled, version, source, created_at, updated_at
		FROM prompt_rules WHERE id = ?`,
		id,
	)
	if err != nil {
		return PromptRule{}, err
	}
	defer rows.Close()
	rules, err := scanPromptRules(rows)
	if err != nil {
		return PromptRule{}, err
	}
	if len(rules) == 0 {
		return PromptRule{}, sql.ErrNoRows
	}
	return rules[0], nil
}

func scanPromptRules(rows *sql.Rows) ([]PromptRule, error) {
	rules := []PromptRule{}
	for rows.Next() {
		var rule PromptRule
		var enabled int
		if err := rows.Scan(&rule.ID, &rule.RuleKey, &rule.Category, &rule.Title, &rule.Content, &enabled, &rule.Version, &rule.Source, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Enabled = enabled == 1
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func defaultPromptRules() []PromptRule {
	return []PromptRule{
		{RuleKey: "truth.no_fabrication", Category: "validation", Title: "No fabrication", Content: "Use only sourced facts. Never invent tools, metrics, titles, leadership, production scope, cloud platforms, or business impact.", Source: "plans/JOB_COPILOT_FULL_PLAN.md"},
		{RuleKey: "truth.weak_matches", Category: "fit_analysis", Title: "Do not force weak matches", Content: "Connect adjacent evidence only when truthful and explicitly mark missing JD details.", Source: "plans/JOB_COPILOT_FULL_PLAN.md"},
		{RuleKey: "jd.top_pain_points", Category: "jd_parse", Title: "Top pain points first", Content: "Prioritize the top 3 employer pain points over exhaustive requirement lists.", Source: "plans/JOB_COPILOT_FULL_PLAN.md"},
		{RuleKey: "jd.ignore_chrome", Category: "jd_parse", Title: "Ignore job-board chrome", Content: "Reject logos, promoted text, profile banners, Premium prompts, application metadata, benefits, salary, location-only rows, and company blurbs.", Source: "current implementation and Reddit resume prompt cautions"},
		{RuleKey: "resume.employer_terms", Category: "resume", Title: "Use employer terminology carefully", Content: "Use JD terminology when it maps to real evidence. Do not copy phrasing word-for-word or add missing tools.", Source: "plans/JOB_COPILOT_PRIORITIES.md"},
		{RuleKey: "resume.human_style", Category: "resume", Title: "Human technical style", Content: "Write concise, specific engineering language. Avoid fake enthusiasm, generic praise, and AI-sounding filler.", Source: "plans/JOB_COPILOT_FULL_PLAN.md"},
		{RuleKey: "fit.blunt_reality", Category: "fit_analysis", Title: "Blunt fit analysis", Content: "Act like a brutally honest hiring fit analyzer. Output an evidence-backed percentage, strengths alignment, critical gaps, reality check, and a directive recommendation. Do not sugar-coat weak matches; most applicants are not competitive when high-priority requirements lack evidence.", Source: "plans/JOB_COPILOT_FULL_PLAN.md + user supplied fit prompt"},
		{RuleKey: "fit.competitive_market", Category: "fit_analysis", Title: "Competitive market calibration", Content: "Assume competitive roles receive many qualified applicants. A partial transferable match is not enough for Apply unless several high-priority needs are directly supported by approved evidence.", Source: "user supplied fit prompt + prompt research synthesis"},
		{RuleKey: "strategy.do_not_overclaim", Category: "strategy", Title: "Do-not-overclaim list", Content: "Carry explicit missing or risky JD concepts into strategy so resume generation avoids exaggeration.", Source: "plans/JOB_COPILOT_PRIORITIES.md"},
		{RuleKey: "prompt.structured_outputs", Category: "prompting", Title: "Structured JSON outputs", Content: "Use explicit schemas, tagged input blocks, compact context, deterministic validation, and fallback paths.", Source: "OpenAI structured output guidance"},
	}
}

func defaultPromptResearchSources() []PromptResearchSource {
	accessed := "2026-06-09"
	return []PromptResearchSource{
		{SourceType: "official_docs", TrustTier: "official", Title: "OpenAI prompt engineering", URL: "https://developers.openai.com/api/docs/guides/prompt-engineering", ExtractedPattern: "Clear task instructions, delimiters, relevant context, and iterative evals.", AppAdaptation: "PromptBuilder uses task/schema/rules/tagged input sections.", AccessedAt: accessed},
		{SourceType: "official_docs", TrustTier: "official", Title: "OpenAI structured outputs", URL: "https://developers.openai.com/api/docs/guides/structured-outputs", ExtractedPattern: "Constrain model output to machine-readable schemas.", AppAdaptation: "JSON-mode generation and strict parsers with local fallback.", AccessedAt: accessed},
		{SourceType: "official_docs", TrustTier: "official", Title: "OpenAI prompt best practices", URL: "https://help.openai.com/en/articles/6654000-guidance-for-writing-effective-prompts", ExtractedPattern: "Be specific, give examples, state output format, and reduce ambiguity.", AppAdaptation: "Rules registry stores compact reusable criteria.", AccessedAt: accessed},
		{SourceType: "community", TrustTier: "forum", Title: "Reddit resume tailoring cautions", URL: "https://www.reddit.com/r/resumes/comments/1lnsrap/chatgpt_for_resume_tailoring/", ExtractedPattern: "Users warn AI often inserts JD-only skills and needs manual review.", AppAdaptation: "Validation rules forbid unsupported tools and require review queues.", AccessedAt: accessed},
		{SourceType: "community", TrustTier: "forum", Title: "Reddit resume prompt structure", URL: "https://www.reddit.com/r/ChatGPTPromptGenius/comments/1ojfel8/prompt_for_resume_tailoring_for_the_job/", ExtractedPattern: "Extract JD keywords, suggest edits with justifications, then draft.", AppAdaptation: "Pipeline separates JD parse, match map, strategy, and bullet drafts.", AccessedAt: accessed},
		{SourceType: "prompt_bank", TrustTier: "prompt_bank", Title: "The Prompt Library resume prompts", URL: "https://www.thepromptlibrary.io/categories/career/resume", ExtractedPattern: "Adapt resume language to job-ad priorities.", AppAdaptation: "Employer terminology rule is gated by evidence matches.", AccessedAt: accessed},
	}
}
