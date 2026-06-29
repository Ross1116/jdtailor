package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type ResumeJSON struct {
	Headline    string            `json:"headline"`
	Summary     string            `json:"summary"`
	ContactLine string            `json:"contact_line"`
	SkillsLine  string            `json:"skills_line"`
	Skills      []ResumeSkill     `json:"skills"`
	Experience  []ResumeEntry     `json:"experience"`
	Projects    []ResumeEntry     `json:"projects"`
	Education   []ResumeEducation `json:"education"`
	TexSource   string            `json:"tex_source"`
	GeneratedAt string            `json:"generated_at"`
}

type ResumeSkill struct {
	Category string   `json:"category"`
	Items    []string `json:"items"`
}

type ResumeEntry struct {
	Company   string   `json:"company"`
	URL       string   `json:"url,omitempty"`
	Title     string   `json:"title"`
	Location  string   `json:"location"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	Bullets   []string `json:"bullets"`
	ClaimIDs  []int64  `json:"claim_ids"`
	BulletIDs []int64  `json:"bullet_ids"`
}

type ResumeEducation struct {
	Organization string `json:"organization"`
	Degree       string `json:"degree"`
	Location     string `json:"location"`
	EndDate      string `json:"end_date"`
}

type ValidationResult struct {
	Passed           bool              `json:"passed"`
	Errors           []string          `json:"errors"`
	Warnings         []string          `json:"warnings"`
	FactualityChecks []FactualityCheck `json:"factuality_checks"`
	StyleIssues      []string          `json:"style_issues"`
	ImmutableIssues  []string          `json:"immutable_issues"`
	TitleIssues      []string          `json:"title_issues"`
}

type FactualityCheck struct {
	BulletIndex int      `json:"bullet_index"`
	Bullet      string   `json:"bullet"`
	HasClaims   bool     `json:"has_claims"`
	AllApproved bool     `json:"all_approved"`
	Issues      []string `json:"issues,omitempty"`
}

type ResumeVersion struct {
	ID               int64            `json:"id"`
	JobID            int64            `json:"job_id"`
	ResumeJSON       ResumeJSON       `json:"resume_json"`
	TexSource        string           `json:"tex_source"`
	PDFPath          string           `json:"pdf_path"`
	ValidationResult ValidationResult `json:"validation_result"`
	CreatedAt        string           `json:"created_at"`
}

type RenderResumeVersionPDFResult struct {
	RenderResult RenderPDFResult `json:"render_result"`
	Version      ResumeVersion   `json:"version"`
}

type Application struct {
	ID                   int64  `json:"id"`
	JobID                int64  `json:"job_id"`
	Status               string `json:"status"`
	FitScore             int    `json:"fit_score"`
	ResumeVersionID      int64  `json:"resume_version_id"`
	CoverLetterVersionID int64  `json:"cover_letter_version_id"`
	Notes                string `json:"notes"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type CorrectionLog struct {
	ID                  int64   `json:"id"`
	ApplicationID       int64   `json:"application_id"`
	ResumeVersionID     int64   `json:"resume_version_id"`
	EntityType          string  `json:"entity_type"`
	FieldPath           string  `json:"field_path"`
	Section             string  `json:"section"`
	EntryIndex          int     `json:"entry_index"`
	ItemIndex           int     `json:"item_index"`
	OriginalBulletText  string  `json:"original_bullet_text"`
	CorrectedBulletText string  `json:"corrected_bullet_text"`
	OriginalText        string  `json:"original_text"`
	CorrectedText       string  `json:"corrected_text"`
	ClaimIDs            []int64 `json:"claim_ids"`
	Reason              string  `json:"reason"`
	CreatedAt           string  `json:"created_at"`
}

type GenerateResumeJSONInput struct {
	JobID             int64   `json:"job_id"`
	SelectedBulletIDs []int64 `json:"selected_bullet_ids"`
}

func (s *Store) GenerateResumeJSON(ctx context.Context, input GenerateResumeJSONInput) (ResumeJSON, error) {
	job, err := s.getJobDescription(input.JobID)
	if err != nil {
		return ResumeJSON{}, err
	}

	analysis, err := s.GetJobAnalysis(input.JobID)
	if errors.Is(err, sql.ErrNoRows) {
		analysis, err = s.AnalyzeJobDescription(input.JobID)
	}
	if err != nil {
		return ResumeJSON{}, err
	}

	strategy, err := s.GetApplicationStrategy(input.JobID)
	if errors.Is(err, sql.ErrNoRows) {
		strategy, err = s.GenerateApplicationStrategy(input.JobID)
	}
	if err != nil {
		return ResumeJSON{}, err
	}

	profile, err := s.GetCandidateProfile()
	if err != nil {
		return ResumeJSON{}, err
	}

	sections, err := s.ListSourceSections(0)
	if err != nil {
		return ResumeJSON{}, fmt.Errorf("list source sections: %w", err)
	}

	approvedFacts, err := s.ListEvidenceFacts("approved")
	if err != nil {
		return ResumeJSON{}, fmt.Errorf("list approved evidence facts: %w", err)
	}

	var selectedDrafts []TailoredBulletDraft
	if len(input.SelectedBulletIDs) > 0 {
		allDrafts, err := s.ListTailoredBulletDrafts(input.JobID)
		if err != nil {
			return ResumeJSON{}, err
		}
		idSet := map[int64]bool{}
		for _, id := range input.SelectedBulletIDs {
			idSet[id] = true
		}
		for _, draft := range allDrafts {
			if idSet[draft.ID] {
				selectedDrafts = append(selectedDrafts, draft)
			}
		}
	} else {
		drafts, listErr := s.ListTailoredBulletDrafts(input.JobID)
		if listErr != nil {
			return ResumeJSON{}, listErr
		}
		for _, draft := range drafts {
			if draft.SelectedForResume {
				selectedDrafts = append(selectedDrafts, draft)
			}
		}
	}

	if len(selectedDrafts) == 0 {
		return ResumeJSON{}, errors.New("no selected bullet drafts for resume generation")
	}
	if len(input.SelectedBulletIDs) == 0 {
		selectedDrafts = balanceSelectedResumeDrafts(selectedDrafts)
	}

	claimIDs := map[int64]bool{}
	for _, draft := range selectedDrafts {
		for _, cid := range draft.ClaimIDs {
			claimIDs[cid] = true
		}
	}

	claims, err := s.ListCandidateClaims("all")
	if err != nil {
		return ResumeJSON{}, err
	}
	claimsByID := map[int64]CandidateClaim{}
	for _, claim := range claims {
		claimsByID[claim.ID] = claim
	}
	// Group experience entries by origin_heading to avoid duplicates
	experienceByKey := map[string]*ResumeEntry{} // origin_heading -> entry pointer
	entryOrder := []string{}
	seenBullets := map[string]bool{}

	for _, draft := range selectedDrafts {
		if seenBullets[draft.DraftText] {
			continue
		}
		seenBullets[draft.DraftText] = true

		originKey := strings.TrimSpace(draft.OriginHeading)

		if entry, exists := experienceByKey[originKey]; exists {
			entry.Bullets = append(entry.Bullets, draft.DraftText)
			entry.ClaimIDs = append(entry.ClaimIDs, draft.ClaimIDs...)
			entry.BulletIDs = append(entry.BulletIDs, draft.ID)
		} else {
			entry := &ResumeEntry{
				Bullets:   []string{draft.DraftText},
				ClaimIDs:  draft.ClaimIDs,
				BulletIDs: []int64{draft.ID},
			}
			originType := strings.TrimSpace(strings.ToLower(draft.OriginType))
			if strings.HasPrefix(originType, "project") {
				entry.Title = draft.OriginHeading
			} else {
				// Parse section metadata for experience entries
				for _, section := range sections {
					if strings.EqualFold(section.Heading, draft.OriginHeading) {
						entry.Company, entry.Location, entry.Title, entry.StartDate, entry.EndDate = parseSectionMetadata(section)
						entry.URL = extractCompanyURL(section)
						break
					}
				}
				// Fallback: if no section found, use the heading as company/title
				if entry.Company == "" {
					entry.Company = draft.OriginHeading
				}
				if entry.Title == "" {
					entry.Title = "Engineer"
				}
			}
			experienceByKey[originKey] = entry
			entryOrder = append(entryOrder, originKey)
		}
	}

	// Split into experience and projects
	experienceEntries := []ResumeEntry{}
	projectEntries := []ResumeEntry{}
	for _, key := range entryOrder {
		entry := experienceByKey[key]
		if entry == nil {
			continue
		}
		isProject := strings.HasPrefix(key, "project") || strings.Contains(strings.ToLower(entry.Title), "project")
		for _, draft := range selectedDrafts {
			if strings.HasPrefix(strings.ToLower(draft.OriginType), "project") || strings.HasPrefix(strings.ToLower(draft.OriginHeading), "project") {
				if strings.EqualFold(draft.OriginHeading, entry.Title) || strings.EqualFold(draft.OriginHeading, key) || strings.EqualFold(draft.OriginHeading, entry.Company) {
					isProject = true
				}
			}
		}
		if isProject {
			projectEntries = append(projectEntries, *entry)
		} else {
			experienceEntries = append(experienceEntries, *entry)
		}
	}
	experienceEntries, projectEntries = supplementResumeEntriesFromSections(experienceEntries, projectEntries, sections)
	sortResumeEntriesBySourceOrder(experienceEntries, sections)
	deduplicateResumeEntryBullets(experienceEntries)
	deduplicateResumeEntryBullets(projectEntries)
	applyHumanToneReview(experienceEntries)
	applyHumanToneReview(projectEntries)
	limitResumeEntries(experienceEntries, 4)
	limitResumeEntries(projectEntries, 2)

	skillSet := map[string]bool{}
	for _, claim := range claims {
		if !claimIDs[claim.ID] && claim.Status != claimStatusApproved && claim.Status != claimStatusApprovedRestricted {
			continue
		}
		for _, tech := range claim.Technologies {
			skillSet[tech] = true
		}
	}
	for _, fact := range approvedFacts {
		for _, tech := range fact.Technologies {
			skillSet[tech] = true
		}
	}

	langSkills := []string{}
	backendSkills := []string{}
	frontendSkills := []string{}
	dbInfraSkills := []string{}
	aiSkills := []string{}
	toolSkills := []string{}
	langSet := map[string]bool{"Go": true, "Python": true, "Java": true, "TypeScript": true, "JavaScript": true, "Rust": true, "C#": true, "C++": true}
	backendSet := map[string]bool{"FastAPI": true, "Gin": true, "Spring Boot": true, "Node.js": true, "Express": true, "REST": true, "gRPC": true, "Microservices": true, "API": true}
	frontendSet := map[string]bool{"React": true, "Next.js": true, "Vite": true, "Tailwind": true}
	dbInfraSet := map[string]bool{"PostgreSQL": true, "MySQL": true, "SQLite": true, "ElasticSearch": true, "Elasticsearch": true, "Redis": true, "MongoDB": true, "Docker": true, "Linux": true, "GitHub Actions": true}
	aiSet := map[string]bool{"LLM": true, "LLM API": true, "OpenAI": true, "Embeddings": true, "Vector Databases": true, "RAG": true}
	jobTerms := resumeTailoringTerms(job, analysis, strategy)
	for skill := range skillSet {
		if !resumeSkillAllowed(skill) {
			continue
		}
		switch {
		case langSet[skill]:
			langSkills = append(langSkills, skill)
		case backendSet[skill]:
			backendSkills = append(backendSkills, skill)
		case frontendSet[skill]:
			frontendSkills = append(frontendSkills, skill)
		case dbInfraSet[skill]:
			dbInfraSkills = append(dbInfraSkills, skill)
		case aiSet[skill]:
			aiSkills = append(aiSkills, skill)
		default:
			if resumeToolSkillAllowed(skill) {
				toolSkills = append(toolSkills, skill)
			}
		}
	}

	skills := []ResumeSkill{}
	if len(langSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Languages", Items: rankResumeSkills(langSkills, jobTerms, 6)})
	}
	if len(backendSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Backend", Items: rankResumeSkills(backendSkills, jobTerms, 7)})
	}
	if len(frontendSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Frontend", Items: rankResumeSkills(frontendSkills, jobTerms, 6)})
	}
	if len(dbInfraSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Databases/Infra", Items: rankResumeSkills(dbInfraSkills, jobTerms, 7)})
	}
	if len(aiSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "AI", Items: rankResumeSkills(aiSkills, jobTerms, 6)})
	}
	if len(toolSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Tools", Items: rankResumeSkills(toolSkills, jobTerms, 5)})
	}

	headline := strings.TrimSpace(profile.Contact.FullName)
	if headline == "" {
		headline = strings.TrimSpace(strategy.ResumeHeadline)
	}
	if headline == "" {
		headline = analysis.RoleTitle
	}

	contactLine := buildContactLine(profile.Contact)
	skillsLine := buildSkillsLine(skills)
	education := buildEducationFromSections(sections)
	summary := buildResumeSummary(job, analysis, strategy, selectedDrafts, claimIDs, claimsByID, approvedFacts, profile, sections)
	resume := ResumeJSON{
		Headline:    headline,
		Summary:     summary,
		ContactLine: contactLine,
		SkillsLine:  skillsLine,
		Skills:      skills,
		Experience:  experienceEntries,
		Projects:    projectEntries,
		Education:   education,
		TexSource:   "",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	resume = applyResumeHumanToneReview(resume)
	resume = optimizeResumeForATS(job, analysis, strategy, resume)
	if edited, editErr := s.HumanizeTailoredResume(ctx, nil, job, analysis, resume); editErr == nil {
		resume = edited
	} else {
		_ = s.LogEvent("warning", "resume human editor skipped: "+editErr.Error())
	}

	return resume, nil
}

func sortResumeEntriesBySourceOrder(entries []ResumeEntry, sections []SourceSection) {
	if len(entries) < 2 || len(sections) == 0 {
		return
	}
	order := map[string]int{}
	for _, section := range sections {
		if !strings.EqualFold(section.SectionType, "experience") {
			continue
		}
		company, _, title, _, _ := parseSectionMetadata(section)
		for _, key := range []string{section.Heading, company, title, company + "|" + title} {
			key = resumeOrderKey(key)
			if key != "" {
				order[key] = section.SortOrder
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		left := resumeEntrySourceOrder(entries[i], order)
		right := resumeEntrySourceOrder(entries[j], order)
		if left == right {
			return false
		}
		return left < right
	})
}

func balanceSelectedResumeDrafts(drafts []TailoredBulletDraft) []TailoredBulletDraft {
	byOrigin := map[string][]TailoredBulletDraft{}
	originOrder := []string{}
	for _, draft := range drafts {
		key := originKey(draft.OriginHeading, draft.OriginType)
		if _, ok := byOrigin[key]; !ok {
			originOrder = append(originOrder, key)
		}
		byOrigin[key] = append(byOrigin[key], draft)
	}
	balanced := []TailoredBulletDraft{}
	for _, key := range originOrder {
		items := byOrigin[key]
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].SelectionScore == items[j].SelectionScore {
				return valueThemeOrder(items[i].ValueTheme) < valueThemeOrder(items[j].ValueTheme)
			}
			return items[i].SelectionScore > items[j].SelectionScore
		})
		budget := bulletBudgetForSectionType(items[0].OriginType)
		coveredThemes := map[string]bool{}
		selected := []TailoredBulletDraft{}
		for _, draft := range items {
			if len(selected) >= budget {
				break
			}
			if len(selected) > 0 && (coveredThemes[normalizeValueTheme(draft.ValueTheme)] || themeLaneAlreadyCovered(draft.ValueTheme, coveredThemes)) {
				continue
			}
			if resumeDraftStoryDuplicate(draft, selected) {
				continue
			}
			selected = append(selected, draft)
			coveredThemes[normalizeValueTheme(draft.ValueTheme)] = true
		}
		for _, draft := range items {
			if len(selected) >= budget {
				break
			}
			if containsDraftID(selected, draft.ID) || resumeDraftStoryDuplicate(draft, selected) {
				continue
			}
			selected = append(selected, draft)
		}
		balanced = append(balanced, selected...)
	}
	return balanced
}

func resumeDraftStoryDuplicate(draft TailoredBulletDraft, selected []TailoredBulletDraft) bool {
	for _, existing := range selected {
		if normalizeValueTheme(draft.ValueTheme) == normalizeValueTheme(existing.ValueTheme) {
			return true
		}
		if themeLaneAlreadyCovered(draft.ValueTheme, map[string]bool{normalizeValueTheme(existing.ValueTheme): true}) && jaccardScore(similarityTokens(draft.DraftText), similarityTokens(existing.DraftText)) >= 0.24 {
			return true
		}
		if storyFamiliesTooSimilar(bulletStoryFamilies(draft.DraftText), bulletStoryFamilies(existing.DraftText)) {
			return true
		}
	}
	return false
}

func containsDraftID(drafts []TailoredBulletDraft, id int64) bool {
	for _, draft := range drafts {
		if draft.ID == id {
			return true
		}
	}
	return false
}

func resumeEntrySourceOrder(entry ResumeEntry, order map[string]int) int {
	keys := []string{entry.Company, entry.Title, entry.Company + "|" + entry.Title}
	for _, key := range keys {
		if value, ok := order[resumeOrderKey(key)]; ok {
			return value
		}
	}
	return 1 << 30
}

func resumeOrderKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func applyResumeHumanToneReview(resume ResumeJSON) ResumeJSON {
	resume.Summary = polishResumeSummary(resume.Summary)
	applyHumanToneReview(resume.Experience)
	applyHumanToneReview(resume.Projects)
	return resume
}

func applyHumanToneReview(entries []ResumeEntry) {
	for i := range entries {
		for j := range entries[i].Bullets {
			entries[i].Bullets[j] = humanizeResumeBullet(entries[i].Bullets[j])
		}
	}
}

func humanizeResumeBullet(text string) string {
	text = normalizeHumanResumeText(text)
	if text == "" {
		return text
	}
	replacements := []struct{ from, to string }{
		{"Python 3.11", "Python"},
		{"python 3.11", "Python"},
		{"transactional backend workflows", "backend logic"},
		{"backend workflows", "backend logic"},
		{"integration workflows", "integrations"},
		{"workflow logic", "application logic"},
		{"workflows", "systems"},
		{"workflow", "system"},
		{"Utilized", "Used"},
		{"utilized", "used"},
		{"Leveraged", "Used"},
		{"leveraged", "used"},
		{"Spearheaded", "Led"},
		{"spearheaded", "led"},
	}
	for _, replacement := range replacements {
		text = strings.ReplaceAll(text, replacement.from, replacement.to)
	}
	return normalizeHumanResumeText(text)
}

func optimizeResumeForATS(job JobDescription, analysis JobAnalysis, strategy ApplicationStrategy, resume ResumeJSON) ResumeJSON {
	terms := resumeTailoringTerms(job, analysis, strategy)
	for i := range resume.Skills {
		resume.Skills[i].Items = rankResumeSkills(resume.Skills[i].Items, terms, 8)
	}
	resume.SkillsLine = buildSkillsLine(resume.Skills)
	return resume
}

func supplementResumeEntriesFromSections(experienceEntries []ResumeEntry, projectEntries []ResumeEntry, sections []SourceSection) ([]ResumeEntry, []ResumeEntry) {
	seenExperience := resumeEntryKeySet(experienceEntries)
	seenProjects := resumeEntryKeySet(projectEntries)
	ordered := append([]SourceSection{}, sections...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].SortOrder < ordered[j].SortOrder })
	for _, section := range ordered {
		sectionType := strings.ToLower(strings.TrimSpace(section.SectionType))
		switch sectionType {
		case "experience":
			entry := resumeEntryFromExperienceSection(section)
			key := resumeEntryKey(entry)
			if key == "" || len(entry.Bullets) == 0 {
				continue
			}
			if seenExperience[key] {
				mergeResumeEntryBullets(experienceEntries, key, entry.Bullets)
				continue
			}
			seenExperience[key] = true
			experienceEntries = append(experienceEntries, entry)
		case "project":
			entry := resumeEntryFromProjectSection(section)
			key := resumeEntryKey(entry)
			if key == "" || len(entry.Bullets) == 0 {
				continue
			}
			if seenProjects[key] {
				mergeResumeEntryBullets(projectEntries, key, entry.Bullets)
				continue
			}
			seenProjects[key] = true
			projectEntries = append(projectEntries, entry)
		}
	}
	return experienceEntries, projectEntries
}

func resumeEntryFromExperienceSection(section SourceSection) ResumeEntry {
	company, location, title, startDate, endDate := parseSectionMetadata(section)
	if company == "" {
		company = section.Heading
	}
	if title == "" {
		title = titleFromHeading(section.Heading)
	}
	return ResumeEntry{
		Company:   company,
		URL:       extractCompanyURL(section),
		Title:     title,
		Location:  location,
		StartDate: startDate,
		EndDate:   endDate,
		Bullets:   limitStrings(extractSectionBullets(section), 3),
	}
}

func resumeEntryFromProjectSection(section SourceSection) ResumeEntry {
	title, detail, startDate, endDate := parseProjectMetadata(section)
	if title == "" {
		title = section.Heading
	}
	return ResumeEntry{
		Title:     title,
		URL:       detail,
		StartDate: startDate,
		EndDate:   endDate,
		Bullets:   limitStrings(extractSectionBullets(section), 2),
	}
}

func resumeEntryKeySet(entries []ResumeEntry) map[string]bool {
	seen := map[string]bool{}
	for _, entry := range entries {
		if key := resumeEntryKey(entry); key != "" {
			seen[key] = true
		}
	}
	return seen
}

func resumeEntryKey(entry ResumeEntry) string {
	return strings.ToLower(strings.TrimSpace(firstNonEmpty(entry.Company, entry.Title)))
}

func limitResumeEntries(entries []ResumeEntry, bulletLimit int) {
	for i := range entries {
		entries[i].Bullets = limitStrings(entries[i].Bullets, bulletLimit)
	}
}

func deduplicateResumeEntryBullets(entries []ResumeEntry) {
	for i := range entries {
		deduped := []string{}
		seen := map[string]bool{}
		for _, bullet := range entries[i].Bullets {
			key := strings.ToLower(strings.TrimSpace(bullet))
			if key == "" || seen[key] || isNearDuplicate(key, seen) {
				continue
			}
			seen[key] = true
			deduped = append(deduped, bullet)
		}
		entries[i].Bullets = deduped
	}
}

func mergeResumeEntryBullets(entries []ResumeEntry, key string, bullets []string) {
	for i := range entries {
		if resumeEntryKey(entries[i]) != key {
			continue
		}
		seen := map[string]bool{}
		for _, bullet := range entries[i].Bullets {
			seen[strings.ToLower(strings.TrimSpace(bullet))] = true
		}
		for _, bullet := range bullets {
			bulletKey := strings.ToLower(strings.TrimSpace(bullet))
			if bulletKey == "" || seen[bulletKey] {
				continue
			}
			if isNearDuplicate(bulletKey, seen) {
				continue
			}
			seen[bulletKey] = true
			entries[i].Bullets = append(entries[i].Bullets, bullet)
		}
		return
	}
}

func isNearDuplicate(text string, existing map[string]bool) bool {
	words := strings.Fields(text)
	if len(words) < 6 {
		return false
	}
	textWords := wordSet(words)
	for prev := range existing {
		prevWords := wordSet(strings.Fields(prev))
		if len(prevWords) == 0 {
			continue
		}
		overlap := 0
		for w := range textWords {
			if prevWords[w] {
				overlap++
			}
		}
		if overlap >= len(prevWords)/2 && overlap >= len(textWords)/2 {
			return true
		}
	}
	return false
}

func wordSet(words []string) map[string]bool {
	seen := map[string]bool{}
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,;:()[]{}"))
		if len(w) > 2 {
			seen[w] = true
		}
	}
	return seen
}

func extractSectionBullets(section SourceSection) []string {
	bullets := []string{}
	for _, line := range nonEmptyLines(section.Content) {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		line = strings.TrimSpace(strings.TrimPrefix(line, `\item`))
		if line == "" || strings.Contains(line, "|") {
			continue
		}
		bullets = append(bullets, strings.Trim(line, "{}"))
	}
	return bullets
}

func parseProjectMetadata(section SourceSection) (title, detail, startDate, endDate string) {
	lines := nonEmptyLines(section.Content)
	if len(lines) == 0 {
		return section.Heading, "", "", ""
	}
	parts := strings.Split(lines[0], "|")
	if len(parts) > 0 {
		title = strings.TrimSpace(parts[0])
	}
	detailParts := []string{}
	seenURLs := map[string]bool{}
	for _, raw := range parts[1:] {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		rangeStart, rangeEnd := splitDateRange(part)
		if rangeStart != "" || rangeEnd != "" {
			startDate, endDate = rangeStart, rangeEnd
			continue
		}
		if url := firstURLInText(part); url != "" {
			label := linkLabelNearURL(part, url)
			detailParts = append(detailParts, `\href{`+url+`}{`+sanitizeLaTeX(label)+`}`)
			seenURLs[url] = true
			continue
		}
		detailParts = append(detailParts, `\emph{`+sanitizeLaTeX(part)+`}`)
	}
	if sectionURL := firstURLInText(section.Heading + "\n" + section.Content); sectionURL != "" && !seenURLs[sectionURL] {
		label := linkLabelNearURL(section.Heading+"\n"+section.Content, sectionURL)
		detailParts = append(detailParts, `\href{`+sectionURL+`}{`+sanitizeLaTeX(label)+`}`)
	}
	if len(detailParts) > 0 {
		detail = ` $|$ ` + strings.Join(detailParts, ` $|$ `)
	}
	return title, detail, startDate, endDate
}

func titleFromHeading(heading string) string {
	for _, sep := range []string{" - ", " | "} {
		parts := strings.Split(heading, sep)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	return ""
}

func resumeSkillAllowed(skill string) bool {
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return false
	}
	lower := strings.ToLower(skill)
	blocked := map[string]bool{
		"sentry":     true,
		"goroutines": true,
		"goroutine":  true,
		"mutexes":    true,
		"mutex":      true,
		"jwt":        true,
		"oauth":      true,
		"rbac":       true,
		"ftp":        true,
		"sftp":       true,
		"node":       true,
	}
	return !blocked[lower]
}

func resumeToolSkillAllowed(skill string) bool {
	lower := strings.ToLower(strings.TrimSpace(skill))
	allowed := map[string]bool{
		"aws":           true,
		"bash":          true,
		"elastic stack": true,
		"sql":           true,
	}
	return allowed[lower]
}

func resumeTailoringTerms(job JobDescription, analysis JobAnalysis, strategy ApplicationStrategy) map[string]bool {
	terms := map[string]bool{}
	add := func(values ...string) {
		for _, value := range values {
			for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
				return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '#'
			}) {
				if len(token) > 1 {
					terms[token] = true
				}
			}
		}
	}
	add(job.Title, job.RawText, analysis.RoleTitle, analysis.RoleArchetype)
	add(analysis.RequiredSkills...)
	add(analysis.PreferredSkills...)
	add(analysis.Responsibilities...)
	add(analysis.TopPainPoints...)
	add(strategy.Keywords...)
	add(strategy.ResumeHeadline, strategy.PositioningStrategy)
	return terms
}

func rankResumeSkills(skills []string, jobTerms map[string]bool, limit int) []string {
	type scoredSkill struct {
		Value string
		Score int
	}
	scored := []scoredSkill{}
	seen := map[string]bool{}
	for _, skill := range normalizeStringList(skills) {
		key := strings.ToLower(strings.TrimSpace(skill))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		score := 0
		for _, token := range strings.FieldsFunc(key, func(r rune) bool { return r == ' ' || r == '-' || r == '/' || r == '.' }) {
			if jobTerms[token] {
				score += 10
			}
		}
		if jobTerms[key] {
			score += 20
		}
		scored = append(scored, scoredSkill{Value: skill, Score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Value < scored[j].Value
		}
		return scored[i].Score > scored[j].Score
	})
	result := []string{}
	for _, item := range scored {
		result = append(result, item.Value)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func buildResumeSummary(job JobDescription, analysis JobAnalysis, strategy ApplicationStrategy, drafts []TailoredBulletDraft, claimIDs map[int64]bool, claimsByID map[int64]CandidateClaim, facts []EvidenceFact, profile CandidateProfile, sections []SourceSection) string {
	techs := []string{}
	capabilities := []string{}
	artifacts := []string{}
	domains := []string{}
	for cid := range claimIDs {
		if claim, ok := claimsByID[cid]; ok {
			techs = append(techs, claim.Technologies...)
			capabilities = append(capabilities, claim.Capabilities...)
			artifacts = append(artifacts, claim.Artifacts...)
			domains = append(domains, claim.Domains...)
		}
	}
	for _, fact := range facts {
		techs = append(techs, fact.Technologies...)
	}
	for _, draft := range drafts {
		artifacts = append(artifacts, humanizeValueTheme(draft.ValueTheme), draft.OriginType)
	}
	techs = normalizeStringList(techs)
	capabilities = normalizeStringList(capabilities)
	artifacts = normalizeStringList(artifacts)
	domains = normalizeStringList(domains)
	jobTerms := resumeTailoringTerms(job, analysis, strategy)
	techs = rankResumeSkills(techs, jobTerms, 6)

	role := humanSummaryRole(analysis.RoleTitle, capabilities, techs)
	focus := humanSummaryFocus(capabilities, domains, artifacts)
	if len(analysis.TopPainPoints) > 0 {
		focus = humanizeRequirementPhrase(analysis.TopPainPoints[0])
	} else if len(analysis.Responsibilities) > 0 {
		focus = humanizeRequirementPhrase(analysis.Responsibilities[0])
	}
	techStr := strings.Join(limitStrings(techs, 6), ", ")
	recentWork := humanSummaryRecentWork(capabilities, artifacts)
	duration := humanExperienceDuration(profile, sections)

	if techStr == "" {
		return polishResumeSummary(role + duration + " building " + focus + ". " + recentWork + ".")
	}
	return polishResumeSummary(role + duration + " building " + focus + ". " + recentWork + ", using " + humanSummaryTechPhrase(techs) + ".")
}

func polishResumeSummary(summary string) string {
	summary = normalizeHumanResumeText(summary)
	replacements := []struct{ from, to string }{
		{"product_platform_delivery", "product delivery"},
		{"backend_api_delivery", "backend API delivery"},
		{"platform_delivery", "platform delivery"},
		{"workflows", "systems"},
		{"workflow", "systems"},
	}
	for _, replacement := range replacements {
		summary = strings.ReplaceAll(summary, replacement.from, replacement.to)
	}
	summary = regexp.MustCompile(`\s+across\s+\d+\s+professional contexts`).ReplaceAllString(summary, "")
	summary = regexp.MustCompile(`(?i)\s+for\s+software engineer\s*-\s*ai/ml`).ReplaceAllString(summary, "")
	summary = regexp.MustCompile(`(?i)\bwith\s+a\s+practical\s+focus\s+on\b`).ReplaceAllString(summary, "building")
	summary = regexp.MustCompile(`(?i)\bcomfortable\s+working\s+across\b`).ReplaceAllString(summary, "using")
	summary = regexp.MustCompile(`(?i)\bwith\s+experience\s+turning\b`).ReplaceAllString(summary, "and turning")
	summary = regexp.MustCompile(`(?i)\bexperience\s+with\s+a\s+practical\s+focus\s+on\s+experience\s+building\s+and\s+operating\b`).ReplaceAllString(summary, "building and operating")
	summary = regexp.MustCompile(`(?i)\bexperience\s+building\s+experience\s+building\s+and\s+operating\b`).ReplaceAllString(summary, "building and operating")
	summary = regexp.MustCompile(`(?i)\bexperience\s+building\s+and\s+operating\b`).ReplaceAllString(summary, "building and operating")
	summary = regexp.MustCompile(`(?i)recent work includes shipping backend product features\.?`).ReplaceAllString(summary, "Experience includes building backend APIs, improving reliability, and turning product requirements into shipped features.")
	return normalizeHumanResumeText(summary)
}

func normalizeHumanResumeText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	replacements := []struct{ from, to string }{
		{" — ", ": "},
		{" – ", ": "},
		{"—", ": "},
		{"–", "-"},
		{"Leveraged", "Used"},
		{"leveraged", "used"},
		{"Utilized", "Used"},
		{"utilized", "used"},
		{"Spearheaded", "Led"},
		{"spearheaded", "led"},
		{"Empowered", "Enabled"},
		{"empowered", "enabled"},
		{"Cutting-edge", "Modern"},
		{"cutting-edge", "modern"},
		{"Seamless", "Reliable"},
		{"seamless", "reliable"},
		{"Dynamic", "Practical"},
		{"dynamic", "practical"},
		{"Transformative", "Useful"},
		{"transformative", "useful"},
		{"game-changer", "improvement"},
		{"business outcomes", "delivery outcomes"},
		{"enhance efficiency", "improve efficiency"},
		{"drive growth", "support growth"},
		{"unlock", "support"},
		{"in-depth", "detailed"},
		{"deep dive", "review"},
	}
	for _, replacement := range replacements {
		text = strings.ReplaceAll(text, replacement.from, replacement.to)
	}
	text = regexp.MustCompile(`(?i)\bas a result\b`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?i)\bin order to\b`).ReplaceAllString(text, "to")
	text = regexp.MustCompile(`(?i)\bit is important to\b`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.ReplaceAll(text, " :", ":")
	text = strings.ReplaceAll(text, " ,", ",")
	text = strings.ReplaceAll(text, " .", ".")
	return strings.TrimSpace(text)
}

func humanizeValueTheme(theme string) string {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case "product_platform_delivery":
		return "shipping backend product features"
	case "backend_api_delivery":
		return "building APIs and backend services"
	case "automation_reliability":
		return "improving reliability through automation"
	case "observability_debugging":
		return "debugging production services"
	}
	cleaned := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(theme)), "_", " ")
	cleaned = strings.Trim(cleaned, ". ")
	return cleaned
}

func humanizeRequirementPhrase(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, ".")
	if text == "" {
		return "shipping practical software systems"
	}
	text = strings.TrimPrefix(text, "Responsible for ")
	text = strings.TrimPrefix(text, "responsible for ")
	text = strings.TrimPrefix(text, "Experience building and operating ")
	text = strings.TrimPrefix(text, "experience building and operating ")
	text = strings.TrimPrefix(text, "Experience with ")
	text = strings.TrimPrefix(text, "experience with ")
	if len(text) > 90 {
		text = text[:90]
		if idx := strings.LastIndex(text, " "); idx > 40 {
			text = text[:idx]
		}
	}
	return strings.ToLower(text[:1]) + text[1:]
}

func humanSummaryTechPhrase(techs []string) string {
	items := normalizeStringList(limitStrings(techs, 4))
	if len(items) == 0 {
		return "the tools needed for production delivery"
	}
	return joinHumanList(items)
}

func humanSummaryRole(roleTitle string, capabilities []string, techs []string) string {
	joined := strings.ToLower(strings.Join(append(capabilities, techs...), " "))
	roleLower := strings.ToLower(roleTitle)
	if strings.Contains(roleLower, "ai") || strings.Contains(roleLower, "ml") || strings.Contains(roleLower, "machine learning") || strings.Contains(roleLower, "llm") {
		if strings.Contains(joined, "python") || strings.Contains(joined, "api") || strings.Contains(joined, "backend") || strings.Contains(joined, "data") {
			return "Software engineer with backend and AI-adjacent product experience"
		}
	}
	backendWords := countBackendWords(joined)
	frontendWords := countFrontendWords(joined)
	if strings.Contains(joined, "backend") || strings.Contains(joined, "api") || strings.Contains(joined, "fastapi") || strings.Contains(joined, "postgres") || strings.Contains(joined, "microservice") || strings.Contains(joined, "golang") || strings.Contains(joined, "go ") {
		if backendWords >= frontendWords && (strings.Contains(joined, "platform") || strings.Contains(joined, "infra") || strings.Contains(joined, "systems")) {
			return "Backend/platform software engineer"
		}
		return "Backend software engineer"
	}
	if backendWords >= frontendWords && backendWords > 0 {
		return "Backend software engineer"
	}
	if strings.Contains(joined, "full stack") || strings.Contains(joined, "react") || strings.Contains(joined, "frontend") {
		return "Frontend/full-stack software engineer"
	}
	if strings.Contains(joined, "data") || strings.Contains(joined, "ml") || strings.Contains(joined, "machine learning") {
		return "Data/platform software engineer"
	}
	return "Software engineer"
}

func countBackendWords(joined string) int {
	return countMatches(joined, "fastapi", "gin", "spring boot", "node.js", "express", "rest", "grpc", "microservice", "api", "postgresql", "mysql", "sqlite", "docker", "go", "golang", "python", "java", "backend", "rbac", "audit", "pipeline", "worker", "job", "sftp", "ftp", "etl", "elastic")
}

func countFrontendWords(joined string) int {
	return countMatches(joined, "react", "next.js", "vite", "tailwind", "frontend", "typescript", "javascript", "angular", "vue", "css", "html", "responsive")
}

func countMatches(text string, terms ...string) int {
	count := 0
	for _, term := range terms {
		if strings.Contains(text, term) {
			count++
		}
	}
	return count
}

func humanSummaryFocus(capabilities []string, domains []string, artifacts []string) string {
	parts := []string{}
	lower := strings.ToLower(strings.Join(append(append([]string{}, capabilities...), artifacts...), " "))
	if strings.Contains(lower, "api") || strings.Contains(lower, "backend") {
		parts = append(parts, "backend APIs")
	}
	if strings.Contains(lower, "data") || strings.Contains(lower, "postgres") || strings.Contains(lower, "model") {
		parts = append(parts, "data-backed product features")
	}
	if strings.Contains(lower, "workflow") || strings.Contains(lower, "worker") || strings.Contains(lower, "automation") {
		parts = append(parts, "automation and reliability")
	}
	if len(parts) == 0 {
		parts = append(parts, limitStrings(append(capabilities, domains...), 2)...)
	}
	if len(parts) == 0 {
		return "practical product engineering"
	}
	return joinHumanList(normalizeStringList(parts))
}

func humanSummaryRecentWork(capabilities []string, artifacts []string) string {
	combined := strings.ToLower(strings.Join(append(append([]string{}, capabilities...), artifacts...), " "))
	work := []string{}
	if strings.Contains(combined, "api") || strings.Contains(combined, "backend") {
		work = append(work, "API design")
	}
	if strings.Contains(combined, "background") || strings.Contains(combined, "worker") || strings.Contains(combined, "job") {
		work = append(work, "background jobs")
	}
	if strings.Contains(combined, "rbac") || strings.Contains(combined, "audit") || strings.Contains(combined, "access") {
		work = append(work, "access control and auditability")
	}
	if strings.Contains(combined, "llm") || strings.Contains(combined, "ai") || strings.Contains(combined, "ingestion") {
		work = append(work, "LLM-assisted ingestion")
	}
	if len(work) == 0 {
		work = append(work, limitStrings(artifacts, 3)...)
	}
	if len(work) == 0 {
		return "Most recent work has been hands-on: building APIs, improving reliability, and supporting production systems"
	}
	return "Recent work has been hands-on: " + joinHumanList(normalizeStringList(work))
}

func joinHumanList(items []string) string {
	items = normalizeStringList(items)
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

func humanExperienceDuration(profile CandidateProfile, sections []SourceSection) string {
	totalMonths := 0
	recordMonths := 0
	for _, record := range profile.Records {
		if !strings.EqualFold(record.RecordType, "experience") {
			continue
		}
		months := dateRangeMonths(record.StartDate, record.EndDate)
		if strings.Contains(strings.ToLower(record.Role+" "+record.Label), "part-time") {
			months /= 2
		}
		recordMonths += months
	}
	if recordMonths > 0 {
		totalMonths = recordMonths
	}
	for _, section := range sections {
		if recordMonths > 0 {
			break
		}
		if !strings.EqualFold(section.SectionType, "experience") {
			continue
		}
		_, _, title, startDate, endDate := parseSectionMetadata(section)
		months := dateRangeMonths(startDate, endDate)
		if strings.Contains(strings.ToLower(title+" "+section.Heading+" "+section.Content), "part-time") {
			months /= 2
		}
		totalMonths += months
	}
	if totalMonths < 12 {
		return ""
	}
	years := totalMonths / 12
	if years <= 1 {
		return " with 1 year of experience"
	}
	return fmt.Sprintf(" with %d+ years of experience", years)
}

func dateRangeMonths(startValue string, endValue string) int {
	startYear := firstYear(startValue)
	if startYear == 0 {
		return 0
	}
	startMonth := firstMonth(startValue)
	if startMonth == 0 {
		startMonth = 1
	}
	endYear := firstYear(endValue)
	endMonth := firstMonth(endValue)
	if endYear == 0 || strings.Contains(strings.ToLower(endValue), "present") || strings.Contains(strings.ToLower(endValue), "current") {
		now := time.Now()
		endYear = now.Year()
		endMonth = int(now.Month())
	}
	if endMonth == 0 {
		endMonth = 12
	}
	months := (endYear-startYear)*12 + endMonth - startMonth + 1
	if months < 0 {
		return 0
	}
	return months
}

func firstYear(value string) int {
	for _, field := range strings.FieldsFunc(value, func(r rune) bool { return r < '0' || r > '9' }) {
		if len(field) != 4 {
			continue
		}
		year, err := strconv.Atoi(field)
		if err == nil && year >= 1970 && year <= time.Now().Year() {
			return year
		}
	}
	return 0
}

func firstMonth(value string) int {
	for _, field := range strings.FieldsFunc(value, func(r rune) bool { return r < '0' || r > '9' }) {
		month, err := strconv.Atoi(field)
		if err == nil && month >= 1 && month <= 12 {
			return month
		}
	}
	return 0
}

func (s *Store) ValidateResumeJSON(resume ResumeJSON, jobID int64) (ValidationResult, error) {
	result := ValidationResult{Passed: true}

	claims, err := s.ListCandidateClaims("all")
	if err != nil {
		return result, err
	}
	claimsByID := map[int64]CandidateClaim{}
	for _, claim := range claims {
		claimsByID[claim.ID] = claim
	}
	draftClaimIDsByID := map[int64][]int64{}
	if jobID > 0 {
		drafts, err := s.ListTailoredBulletDrafts(jobID)
		if err != nil {
			return result, err
		}
		for _, draft := range drafts {
			draftClaimIDsByID[draft.ID] = append([]int64{}, draft.ClaimIDs...)
		}
	}

	profile, err := s.GetCandidateProfile()
	if err != nil {
		return result, err
	}

	var allChecks []FactualityCheck
	type resumeBulletValidation struct {
		text     string
		claimIDs []int64
	}
	allBullets := []string{}
	bulletValidations := []resumeBulletValidation{}
	for _, entry := range resume.Experience {
		allBullets = append(allBullets, entry.Bullets...)
		for i, bullet := range entry.Bullets {
			claimIDs := entry.ClaimIDs
			if i < len(entry.BulletIDs) {
				if scopedClaimIDs, ok := draftClaimIDsByID[entry.BulletIDs[i]]; ok {
					claimIDs = scopedClaimIDs
				}
			}
			bulletValidations = append(bulletValidations, resumeBulletValidation{text: bullet, claimIDs: claimIDs})
		}
	}
	for _, entry := range resume.Projects {
		allBullets = append(allBullets, entry.Bullets...)
		for i, bullet := range entry.Bullets {
			claimIDs := entry.ClaimIDs
			if i < len(entry.BulletIDs) {
				if scopedClaimIDs, ok := draftClaimIDsByID[entry.BulletIDs[i]]; ok {
					claimIDs = scopedClaimIDs
				}
			}
			bulletValidations = append(bulletValidations, resumeBulletValidation{text: bullet, claimIDs: claimIDs})
		}
	}

	for i, bulletValidation := range bulletValidations {
		bullet := bulletValidation.text
		check := FactualityCheck{
			BulletIndex: i,
			Bullet:      bullet,
			HasClaims:   len(bulletValidation.claimIDs) > 0,
			AllApproved: true,
		}

		lower := strings.ToLower(bullet)
		evidence := strings.ToLower(supportedBulletEvidence(bulletValidation.claimIDs, claimsByID))
		linkedSensitiveTerms := linkedSensitiveTerms(lower)
		for _, claimID := range bulletValidation.claimIDs {
			claim, ok := claimsByID[claimID]
			if !ok {
				check.Issues = append(check.Issues, fmt.Sprintf("linked claim %d not found", claimID))
				if len(linkedSensitiveTerms) > 0 {
					check.AllApproved = false
				}
				continue
			}
			switch claim.Status {
			case claimStatusApproved:
			case claimStatusApprovedRestricted:
				check.Issues = append(check.Issues, fmt.Sprintf("linked claim %d is approved with restrictions; verify context", claimID))
			case claimStatusRejected, claimStatusBlocked:
				check.Issues = append(check.Issues, fmt.Sprintf("linked claim %d is %s and cannot support resume text", claimID, claim.Status))
				check.AllApproved = false
			default:
				check.Issues = append(check.Issues, fmt.Sprintf("linked claim %d is %s, not approved", claimID, firstNonEmpty(claim.Status, "unreviewed")))
				if len(linkedSensitiveTerms) > 0 {
					check.AllApproved = false
				}
			}
		}
		for _, term := range linkedSensitiveTerms {
			if !strings.Contains(evidence, term) {
				check.Issues = append(check.Issues, "unsupported sensitive term: "+term)
				check.AllApproved = false
			}
		}

		if !check.HasClaims {
			if len(linkedSensitiveTerms) > 0 {
				check.Issues = append(check.Issues, "sensitive terms require linked approved evidence: "+strings.Join(linkedSensitiveTerms, ", "))
				check.AllApproved = false
			} else {
				check.Issues = append(check.Issues, "no linked claims; verify content is evidence-backed")
			}
		}

		allChecks = append(allChecks, check)
	}
	result.FactualityChecks = allChecks

	styleIssues := []string{}
	lowerBullets := strings.ToLower(strings.Join(allBullets, " "))
	bannedPhrases := []string{
		"leveraged", "spearheaded", "empowered", "utilized", "cutting-edge",
		"game-changer", "transformative", "synergy", "seamless",
	}
	for _, phrase := range bannedPhrases {
		if strings.Contains(lowerBullets, phrase) {
			styleIssues = append(styleIssues, "banned phrase: "+phrase)
		}
	}
	result.StyleIssues = styleIssues

	immutableIssues := []string{}
	if profile.Contact.FullName != "" {
		nameParts := strings.Fields(profile.Contact.FullName)
		for _, part := range nameParts {
			partLower := strings.ToLower(part)
			if !strings.Contains(strings.ToLower(resume.Headline), partLower) && !strings.Contains(strings.ToLower(resume.Summary), partLower) {
				immutableIssues = append(immutableIssues, "candidate name part missing from headline and summary: "+part)
				break
			}
		}
	}
	result.ImmutableIssues = immutableIssues

	titleIssues := []string{}
	if strings.TrimSpace(resume.Headline) == "" {
		titleIssues = append(titleIssues, "headline is empty")
	}
	for _, entry := range resume.Experience {
		if strings.TrimSpace(entry.Title) == "" {
			titleIssues = append(titleIssues, "experience entry missing title")
		}
	}
	result.TitleIssues = titleIssues

	for _, entry := range resume.Experience {
		for _, bullet := range entry.Bullets {
			bulletIssues := styleRiskFlags(bullet)
			for _, issue := range bulletIssues {
				result.Warnings = append(result.Warnings, "bullet style: "+issue+": "+bullet)
			}
		}
	}

	for _, check := range result.FactualityChecks {
		if !check.AllApproved {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("bullet %d: unapproved claims", check.BulletIndex))
		}
		for _, issue := range check.Issues {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bullet %d: %s", check.BulletIndex, issue))
		}
	}

	for _, issue := range result.StyleIssues {
		result.Warnings = append(result.Warnings, issue)
	}
	for _, issue := range result.ImmutableIssues {
		result.Errors = append(result.Errors, issue)
		result.Passed = false
	}
	for _, issue := range result.TitleIssues {
		result.Errors = append(result.Errors, issue)
		result.Passed = false
	}

	return result, nil
}

func (s *Store) HumanizeTailoredResume(ctx context.Context, client *http.Client, job JobDescription, analysis JobAnalysis, resume ResumeJSON) (ResumeJSON, error) {
	atsTerms := rankResumeSkills(append(append([]string{}, analysis.RequiredSkills...), analysis.PreferredSkills...), resumeTailoringTerms(job, analysis, ApplicationStrategy{}), 18)
	input := map[string]any{
		"job_title":       job.Title,
		"company":         job.Company,
		"jd_focus":        append(analysis.TopPainPoints, analysis.RequiredSkills...),
		"ats_terms":       atsTerms,
		"resume":          resume,
		"editable_fields": []string{"summary", "skills", "skills_line", "experience.bullets", "projects.bullets"},
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return resume, err
	}
	system := `You are JD Tailor's final human resume editor. You review the output from the JD analyst, evidence matcher, bullet drafter, and resume assembler. Return only valid JSON for the same resume schema.
Rules:
- Make the summary, skills ordering, and bullets sound human, precise, and professional.
- Tailor wording toward the JD's actual needs.
- Optimize for ATS by placing supported JD keywords naturally in Summary, Skills, and relevant bullets.
- Prefer exact JD phrasing for supported tools/responsibilities, but never stuff keywords or make awkward sentences.
- If two JDs are different, the summary and bullets should visibly emphasize different supported themes.
- Do not copy the JD role title into the summary. Avoid patterns like "for Software Engineer - AI/ML".
- Write the summary as 2 natural sentences about the candidate's evidence-backed strengths and the target role's relevant needs.
- Do not use stock phrases like "Recent work includes" or internal labels such as product_platform_delivery.
- Do not invent tools, metrics, companies, titles, domains, responsibilities, dates, or impact.
- Preserve contact, education, company/title/date fields, claim_ids, and bullet_ids exactly.
- Preserve the exact order of experience entries, project entries, and bullets. Edit wording in place only.
- Avoid AI resume cliches: leveraged, spearheaded, dynamic, innovative, cutting-edge, seamless, empowered, utilized.
- Keep concise one-page resume style.`
	text, err := s.GenerateLLMText(ctx, client, system, string(payload), 2600)
	if err != nil {
		return resume, err
	}
	var edited ResumeJSON
	if err := json.Unmarshal([]byte(extractJSONObject(text)), &edited); err != nil {
		return resume, err
	}
	edited = normalizeResumeJSONForHumanEdit(resume, edited)
	return edited, nil
}

func normalizeResumeJSONForHumanEdit(original ResumeJSON, edited ResumeJSON) ResumeJSON {
	edited.ContactLine = original.ContactLine
	edited.Headline = firstNonEmpty(edited.Headline, original.Headline)
	edited.Summary = polishResumeSummary(firstNonEmpty(edited.Summary, original.Summary))
	edited.TexSource = original.TexSource
	edited.GeneratedAt = original.GeneratedAt
	if len(edited.Experience) != len(original.Experience) {
		edited.Experience = original.Experience
	} else {
		for i := range edited.Experience {
			edited.Experience[i].Company = original.Experience[i].Company
			edited.Experience[i].URL = original.Experience[i].URL
			edited.Experience[i].Title = original.Experience[i].Title
			edited.Experience[i].Location = original.Experience[i].Location
			edited.Experience[i].StartDate = original.Experience[i].StartDate
			edited.Experience[i].EndDate = original.Experience[i].EndDate
			edited.Experience[i].ClaimIDs = original.Experience[i].ClaimIDs
			edited.Experience[i].BulletIDs = original.Experience[i].BulletIDs
			if len(edited.Experience[i].Bullets) != len(original.Experience[i].Bullets) {
				edited.Experience[i].Bullets = original.Experience[i].Bullets
			} else {
				for j := range edited.Experience[i].Bullets {
					edited.Experience[i].Bullets[j] = humanizeResumeBullet(edited.Experience[i].Bullets[j])
				}
			}
		}
	}
	if len(edited.Projects) != len(original.Projects) {
		edited.Projects = original.Projects
	} else {
		for i := range edited.Projects {
			edited.Projects[i].Company = original.Projects[i].Company
			edited.Projects[i].URL = original.Projects[i].URL
			edited.Projects[i].Title = original.Projects[i].Title
			edited.Projects[i].Location = original.Projects[i].Location
			edited.Projects[i].StartDate = original.Projects[i].StartDate
			edited.Projects[i].EndDate = original.Projects[i].EndDate
			edited.Projects[i].ClaimIDs = original.Projects[i].ClaimIDs
			edited.Projects[i].BulletIDs = original.Projects[i].BulletIDs
			if len(edited.Projects[i].Bullets) != len(original.Projects[i].Bullets) {
				edited.Projects[i].Bullets = original.Projects[i].Bullets
			} else {
				for j := range edited.Projects[i].Bullets {
					edited.Projects[i].Bullets[j] = humanizeResumeBullet(edited.Projects[i].Bullets[j])
				}
			}
		}
	}
	if len(edited.Education) == 0 {
		edited.Education = original.Education
	}
	if len(edited.Skills) == 0 {
		edited.Skills = original.Skills
	}
	if edited.SkillsLine == "" {
		edited.SkillsLine = buildSkillsLine(edited.Skills)
	}
	if edited.Summary == "" {
		edited.Summary = original.Summary
	}
	return edited
}

func extractJSONObject(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		return text
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func supportedBulletEvidence(claimIDs []int64, claimsByID map[int64]CandidateClaim) string {
	parts := []string{}
	for _, claimID := range claimIDs {
		claim, ok := claimsByID[claimID]
		if !ok || (claim.Status != claimStatusApproved && claim.Status != claimStatusApprovedRestricted) {
			continue
		}
		parts = append(parts,
			claim.ClaimText,
			strings.Join(claim.Actions, " "),
			strings.Join(claim.Technologies, " "),
			strings.Join(claim.Capabilities, " "),
			strings.Join(claim.Objects, " "),
			strings.Join(claim.Domains, " "),
			strings.Join(claim.Artifacts, " "),
			strings.Join(claim.Scope, " "),
			strings.Join(claim.Metrics, " "),
			strings.Join(claim.Outcomes, " "),
		)
	}
	return strings.Join(parts, " ")
}

func linkedSensitiveTerms(lower string) []string {
	terms := []string{}
	for _, term := range []string{"aws", "serverless", "container", "containers", "kubernetes", "health-tech", "healthcare", "medical", "compliance", "enterprise", "scalable"} {
		if containsNormalizedTerm(lower, term) {
			terms = append(terms, term)
		}
	}
	return terms
}

func (s *Store) RenderResumePDF(ctx context.Context, resume ResumeJSON) (RenderPDFResult, error) {
	return s.renderResumePDF(ctx, resume, fmt.Sprintf("resume-%s", renderTimestamp()))
}

func (s *Store) RenderResumeVersionPDF(ctx context.Context, versionID int64) (RenderResumeVersionPDFResult, error) {
	version, err := s.GetResumeVersion(versionID)
	if err != nil {
		return RenderResumeVersionPDFResult{}, err
	}
	renderResult, err := s.renderResumePDF(ctx, version.ResumeJSON, fmt.Sprintf("resume-version-%d-%s", versionID, renderTimestamp()))
	if err != nil {
		return RenderResumeVersionPDFResult{}, err
	}
	if !renderResult.Success {
		return RenderResumeVersionPDFResult{RenderResult: renderResult, Version: version}, nil
	}
	updated, err := s.UpdateResumeVersionRenderMetadata(version.ID, readRenderedTex(renderResult.TexPath), renderResult.PDFPath)
	if err != nil {
		return RenderResumeVersionPDFResult{}, err
	}
	return RenderResumeVersionPDFResult{RenderResult: renderResult, Version: updated}, nil
}

func (s *Store) renderResumePDF(ctx context.Context, resume ResumeJSON, outputDirName string) (RenderPDFResult, error) {
	texContent, err := renderResumeTemplate(resume)
	if err != nil {
		return RenderPDFResult{}, err
	}
	texContent = hardenResumeTex(texContent)

	firstName := resumeFirstName(resume.Headline)
	if firstName == "" {
		firstName = "resume"
	}
	pdfName := firstName + "_Resume"

	outputDir := filepath.Join(s.generatedPath, outputDirName)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return RenderPDFResult{}, err
	}

	texPath := filepath.Join(outputDir, pdfName+".tex")
	if err := os.WriteFile(texPath, []byte(texContent), 0o644); err != nil {
		return RenderPDFResult{}, err
	}

	pdfPath := filepath.Join(outputDir, pdfName+".pdf")
	status := s.TectonicStatus()
	pdfResult := RenderPDFResult{TexPath: texPath, PDFPath: pdfPath, OutputDir: outputDir}
	if status.Status != "installed" {
		pdfResult.Error = "Tectonic is " + status.Status
		_ = s.LogEvent("error", "resume PDF render failed: "+pdfResult.Error)
		return pdfResult, nil
	}
	cmd := execCommandContext(ctx, status.ExecutablePath, "-X", "compile", "--outdir", outputDir, texPath)
	cmd.Dir = outputDir
	output, pdfErr := cmd.CombinedOutput()
	if pdfErr != nil {
		pdfResult.Error = strings.TrimSpace(string(output))
		if pdfResult.Error == "" {
			pdfResult.Error = pdfErr.Error()
		}
		_ = s.LogEvent("error", "resume PDF render failed: "+pdfResult.Error)
		return pdfResult, nil
	}
	if _, err := os.Stat(pdfPath); err != nil {
		pdfResult.Error = "PDF was not created"
		_ = s.LogEvent("error", "resume PDF render failed: "+pdfResult.Error)
		return pdfResult, nil
	}
	pdfResult.Success = true
	_ = s.LogEvent("info", "resume PDF rendered")

	return RenderPDFResult{
		Success:   pdfResult.Success,
		TexPath:   texPath,
		PDFPath:   pdfPath,
		OutputDir: outputDir,
		Error:     pdfResult.Error,
	}, nil
}

func renderTimestamp() string {
	return time.Now().UTC().Format("20060102T150405.000000000Z")
}

func readRenderedTex(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func resumeFirstName(headline string) string {
	headline = strings.TrimSpace(headline)
	if headline == "" {
		return ""
	}
	idx := strings.IndexAny(headline, " ,")
	if idx <= 0 {
		return headline
	}
	return strings.TrimSpace(headline[:idx])
}

func normalizeResumeTexSource(source string) string {
	tex := strings.TrimSpace(source)
	if tex == "" {
		return ""
	}
	if strings.HasPrefix(tex, "```") {
		tex = strings.TrimPrefix(tex, "```latex")
		tex = strings.TrimPrefix(tex, "```tex")
		tex = strings.TrimPrefix(tex, "```")
		tex = strings.TrimSuffix(tex, "```")
		tex = strings.TrimSpace(tex)
	}
	if idx := strings.Index(tex, `\documentclass`); idx >= 0 {
		tex = tex[idx:]
	}
	if end := strings.LastIndex(tex, `\end{document}`); end >= 0 {
		tex = tex[:end+len(`\end{document}`)]
	}
	return strings.TrimSpace(tex)
}

func validResumeTexDocument(tex string) bool {
	trimmed := strings.TrimSpace(tex)
	return strings.HasPrefix(trimmed, `\documentclass`) && strings.Contains(trimmed, `\begin{document}`) && strings.Contains(trimmed, `\end{document}`)
}

func hardenResumeTex(tex string) string {
	lines := strings.Split(tex, "\n")
	kept := make([]string, 0, len(lines))
	skipNextFi := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if skipNextFi && trimmed == `\fi` {
			skipNextFi = false
			continue
		}
		if strings.Contains(trimmed, `\ifdefined\pdfgentounicode`) {
			skipNextFi = true
			continue
		}
		if strings.Contains(trimmed, "glyphtounicode") || strings.Contains(trimmed, `\pdfgentounicode`) {
			continue
		}
		kept = append(kept, line)
	}
	tex = strings.Join(kept, "\n")
	return repairSkillsItemize(tex)
}

func repairSkillsItemize(tex string) string {
	section := `\section{Technical Skills}`
	sectionIdx := strings.Index(tex, section)
	if sectionIdx < 0 {
		return tex
	}
	startRel := strings.Index(tex[sectionIdx:], `\resumeSubHeadingListStart`)
	if startRel < 0 {
		return tex
	}
	start := sectionIdx + startRel
	endRel := strings.Index(tex[start:], `\resumeSubHeadingListEnd`)
	if endRel < 0 {
		return tex
	}
	end := start + endRel
	block := tex[start:end]
	if strings.Contains(block, `\item`) || !strings.Contains(block, `\textbf`) {
		return tex
	}
	inner := strings.TrimSpace(strings.TrimPrefix(block, `\resumeSubHeadingListStart`))
	repaired := `\resumeSubHeadingListStart
\small{\item{
` + inner + `
}}
`
	return tex[:start] + repaired + tex[end:]
}

func renderResumeTemplate(resume ResumeJSON) (string, error) {
	funcs := template.FuncMap{
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"tex": sanitizeLaTeX,
	}
	tmpl, err := template.New("resume").Delims("[[", "]]").Funcs(funcs).Parse(latexResumeTemplate())
	if err != nil {
		return "", fmt.Errorf("failed to parse LaTeX template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, resume); err != nil {
		return "", fmt.Errorf("failed to execute LaTeX template: %w", err)
	}
	return buf.String(), nil
}

func (s *Store) SaveResumeVersion(version ResumeVersion) (ResumeVersion, error) {
	if version.JobID <= 0 {
		return ResumeVersion{}, fmt.Errorf("invalid job_id %d for resume version", version.JobID)
	}
	exists, err := s.jobExists(version.JobID)
	if err != nil {
		return ResumeVersion{}, fmt.Errorf("checking job existence: %w", err)
	}
	if !exists {
		return ResumeVersion{}, fmt.Errorf("job %d does not exist", version.JobID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	resumeJSONBytes, _ := json.Marshal(version.ResumeJSON)
	validationBytes, _ := json.Marshal(version.ValidationResult)

	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO resume_versions (job_id, resume_json, tex_source, pdf_path, validation_result, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		version.JobID,
		string(resumeJSONBytes),
		version.TexSource,
		version.PDFPath,
		string(validationBytes),
		now,
	)
	if err != nil {
		return ResumeVersion{}, err
	}
	id, _ := result.LastInsertId()
	version.ID = id
	version.CreatedAt = now
	return version, nil
}

func (s *Store) UpdateResumeVersionRenderMetadata(id int64, texSource string, pdfPath string) (ResumeVersion, error) {
	if id <= 0 {
		return ResumeVersion{}, fmt.Errorf("invalid resume version id %d", id)
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE resume_versions SET tex_source = ?, pdf_path = ? WHERE id = ?`,
		texSource,
		pdfPath,
		id,
	)
	if err != nil {
		return ResumeVersion{}, err
	}
	return s.GetResumeVersion(id)
}

func (s *Store) GetResumeVersion(id int64) (ResumeVersion, error) {
	var version ResumeVersion
	var resumeJSONStr, validationStr string
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, job_id, resume_json, tex_source, pdf_path, validation_result, created_at
		FROM resume_versions WHERE id = ?`,
		id,
	).Scan(&version.ID, &version.JobID, &resumeJSONStr, &version.TexSource, &version.PDFPath, &validationStr, &version.CreatedAt)
	if err != nil {
		return ResumeVersion{}, err
	}
	if err := json.Unmarshal([]byte(resumeJSONStr), &version.ResumeJSON); err != nil {
		return ResumeVersion{}, fmt.Errorf("decode resume_json: %w", err)
	}
	if err := json.Unmarshal([]byte(validationStr), &version.ValidationResult); err != nil {
		return ResumeVersion{}, fmt.Errorf("decode validation_result: %w", err)
	}
	return version, nil
}

func (s *Store) ListResumeVersions(jobID int64) ([]ResumeVersion, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, resume_json, tex_source, pdf_path, validation_result, created_at
		FROM resume_versions WHERE job_id = ? ORDER BY created_at DESC`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []ResumeVersion
	for rows.Next() {
		var version ResumeVersion
		var resumeJSONStr, validationStr string
		if err := rows.Scan(&version.ID, &version.JobID, &resumeJSONStr, &version.TexSource, &version.PDFPath, &validationStr, &version.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(resumeJSONStr), &version.ResumeJSON); err != nil {
			return nil, fmt.Errorf("decode resume_json for version %d: %w", version.ID, err)
		}
		if err := json.Unmarshal([]byte(validationStr), &version.ValidationResult); err != nil {
			return nil, fmt.Errorf("decode validation_result for version %d: %w", version.ID, err)
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) SaveApplication(app Application) (Application, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if app.ID > 0 {
		_, err := s.db.ExecContext(
			context.Background(),
			`UPDATE applications SET status = ?, fit_score = ?, resume_version_id = ?, cover_letter_version_id = ?, notes = ?, updated_at = ? WHERE id = ?`,
			normalizeAppStatus(app.Status), app.FitScore, app.ResumeVersionID, app.CoverLetterVersionID, strings.TrimSpace(app.Notes), now, app.ID,
		)
		if err != nil {
			return Application{}, err
		}
		app.UpdatedAt = now
		return app, nil
	}
	if app.JobID <= 0 {
		return Application{}, fmt.Errorf("invalid job_id %d for application", app.JobID)
	}

	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO applications (job_id, status, fit_score, resume_version_id, cover_letter_version_id, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		app.JobID, normalizeAppStatus(app.Status), app.FitScore, app.ResumeVersionID, app.CoverLetterVersionID, strings.TrimSpace(app.Notes), now, now,
	)
	if err != nil {
		return Application{}, err
	}
	id, _ := result.LastInsertId()
	app.ID = id
	app.CreatedAt = now
	app.UpdatedAt = now
	return app, nil
}

func (s *Store) GetApplication(id int64) (Application, error) {
	var app Application
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, job_id, status, fit_score, resume_version_id, cover_letter_version_id, notes, created_at, updated_at
		FROM applications WHERE id = ?`,
		id,
	).Scan(&app.ID, &app.JobID, &app.Status, &app.FitScore, &app.ResumeVersionID, &app.CoverLetterVersionID, &app.Notes, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return Application{}, err
	}
	return app, nil
}

func (s *Store) ListApplications() ([]Application, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, status, fit_score, resume_version_id, cover_letter_version_id, notes, created_at, updated_at
		FROM applications ORDER BY updated_at DESC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var app Application
		if err := rows.Scan(&app.ID, &app.JobID, &app.Status, &app.FitScore, &app.ResumeVersionID, &app.CoverLetterVersionID, &app.Notes, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (s *Store) UpdateApplicationStatus(id int64, status string) (Application, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(context.Background(), `UPDATE applications SET status = ?, updated_at = ? WHERE id = ?`, normalizeAppStatus(status), now, id)
	if err != nil {
		return Application{}, err
	}
	return s.GetApplication(id)
}

func (s *Store) UpdateApplicationResumeVersion(id int64, resumeVersionID int64) (Application, error) {
	app, err := s.GetApplication(id)
	if err != nil {
		return Application{}, err
	}
	version, err := s.GetResumeVersion(resumeVersionID)
	if err != nil {
		return Application{}, err
	}
	if version.JobID != app.JobID {
		return Application{}, fmt.Errorf("resume version %d belongs to job %d, not application job %d", resumeVersionID, version.JobID, app.JobID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(context.Background(), `UPDATE applications SET resume_version_id = ?, updated_at = ? WHERE id = ?`, resumeVersionID, now, id)
	if err != nil {
		return Application{}, err
	}
	return s.GetApplication(id)
}

func (s *Store) LogCorrection(correction CorrectionLog) (CorrectionLog, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	claimJSON, _ := encodeInt64List(correction.ClaimIDs)
	correction = normalizeCorrectionLog(correction)
	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO correction_logs (application_id, resume_version_id, entity_type, field_path, section, entry_index, item_index, original_text, corrected_text, claim_ids_json, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		correction.ApplicationID,
		correction.ResumeVersionID,
		correction.EntityType,
		correction.FieldPath,
		correction.Section,
		correction.EntryIndex,
		correction.ItemIndex,
		correction.OriginalText,
		correction.CorrectedText,
		claimJSON,
		strings.TrimSpace(correction.Reason),
		now,
	)
	if err != nil {
		return CorrectionLog{}, err
	}
	id, _ := result.LastInsertId()
	correction.ID = id
	correction.CreatedAt = now
	return correction, nil
}

func (s *Store) ListCorrections(applicationID int64) ([]CorrectionLog, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, application_id, resume_version_id, entity_type, field_path, section, entry_index, item_index, original_text, corrected_text, claim_ids_json, reason, created_at
		FROM correction_logs WHERE application_id = ? ORDER BY created_at DESC`,
		applicationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var corrections []CorrectionLog
	for rows.Next() {
		var c CorrectionLog
		var claimJSON string
		if err := rows.Scan(&c.ID, &c.ApplicationID, &c.ResumeVersionID, &c.EntityType, &c.FieldPath, &c.Section, &c.EntryIndex, &c.ItemIndex, &c.OriginalText, &c.CorrectedText, &claimJSON, &c.Reason, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ClaimIDs = decodeInt64List(claimJSON)
		c = normalizeCorrectionLog(c)
		corrections = append(corrections, c)
	}
	return corrections, rows.Err()
}

func normalizeCorrectionLog(c CorrectionLog) CorrectionLog {
	c.EntityType = firstNonEmpty(strings.TrimSpace(c.EntityType), "resume_bullet")
	c.FieldPath = strings.TrimSpace(c.FieldPath)
	c.Section = strings.TrimSpace(c.Section)
	if c.EntryIndex == 0 && !fieldPathHasIndex(c.FieldPath, "experience", "projects", "education", "skills") {
		c.EntryIndex = -1
	}
	if c.ItemIndex == 0 && !fieldPathHasIndex(c.FieldPath, "bullets", "items") {
		c.ItemIndex = -1
	}
	c.OriginalText = firstNonEmpty(c.OriginalText, c.OriginalBulletText)
	c.CorrectedText = firstNonEmpty(c.CorrectedText, c.CorrectedBulletText)
	c.OriginalBulletText = firstNonEmpty(c.OriginalBulletText, c.OriginalText)
	c.CorrectedBulletText = firstNonEmpty(c.CorrectedBulletText, c.CorrectedText)
	c.Reason = strings.TrimSpace(c.Reason)
	return c
}

func fieldPathHasIndex(fieldPath string, names ...string) bool {
	for _, name := range names {
		if regexp.MustCompile(regexp.QuoteMeta(name) + `\[\d+\]`).MatchString(fieldPath) {
			return true
		}
	}
	return false
}

func normalizeAppStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "ready_to_apply":
		return "draft"
	case "draft", "applied", "rejected", "interviewing", "offer":
		return strings.TrimSpace(strings.ToLower(status))
	}
	return "draft"
}

func buildContactLine(contact CandidateContact) string {
	parts := []string{}
	if contact.Phone != "" {
		parts = append(parts, "\\href{tel:"+contact.Phone+"}{"+sanitizeLaTeX(contact.Phone)+"}")
	}
	if contact.Email != "" {
		parts = append(parts, "\\href{mailto:"+contact.Email+"}{"+sanitizeLaTeX(contact.Email)+"}")
	}
	if contact.LinkedIn != "" {
		parts = append(parts, "\\href{https://"+strings.TrimPrefix(contact.LinkedIn, "https://")+"}{"+sanitizeLaTeX(contact.LinkedIn)+"}")
	}
	if contact.GitHub != "" {
		parts = append(parts, "\\href{https://"+strings.TrimPrefix(contact.GitHub, "https://")+"}{"+sanitizeLaTeX(contact.GitHub)+"}")
	}
	if contact.Location != "" {
		parts = append(parts, sanitizeLaTeX(contact.Location))
	}
	return strings.Join(parts, " $|$ ")
}

func buildSkillsLine(skills []ResumeSkill) string {
	parts := []string{}
	for _, skill := range skills {
		parts = append(parts, skill.Category+": "+strings.Join(skill.Items, ", "))
	}
	return strings.Join(parts, " \\ ")
}

func buildEducationFromSections(sections []SourceSection) []ResumeEducation {
	education := []ResumeEducation{}
	for _, section := range sections {
		if section.SectionType != "education" {
			continue
		}
		entry := parseEducationSection(section)
		if entry.Degree == "" && entry.Location == "" && entry.Organization == "" {
			continue
		}
		education = append(education, entry)
	}
	if len(education) == 0 {
		for _, section := range sections {
			heading := strings.TrimSpace(section.Heading)
			content := strings.TrimSpace(section.Content)
			if heading == "" && content == "" {
				continue
			}
			lower := strings.ToLower(heading + " " + content)
			if strings.Contains(lower, "university") || strings.Contains(lower, "bachelor") || strings.Contains(lower, "master") || strings.Contains(lower, "phd") || strings.Contains(lower, "college") || strings.Contains(lower, "institute") {
				education = append(education, ResumeEducation{
					Organization: heading,
					Degree:       content,
				})
			}
		}
	}
	return education
}

func parseEducationSection(section SourceSection) ResumeEducation {
	entry := ResumeEducation{}
	heading := strings.TrimSpace(section.Heading)
	if left, right, ok := strings.Cut(heading, " - "); ok {
		entry.Organization = strings.TrimSpace(left)
		if educationDegreeLike(right) {
			entry.Degree = strings.TrimSpace(right)
		}
	} else if educationDegreeLike(heading) {
		entry.Degree = heading
	} else {
		entry.Organization = heading
	}

	for _, line := range nonEmptyLines(section.Content) {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line == "" {
			continue
		}
		for _, field := range educationFields(line) {
			applyEducationField(&entry, strings.TrimSpace(field))
		}
	}
	if entry.Location == "" {
		entry.Location = extractEducationLocation(section.Content)
	}
	return entry
}

func applyEducationField(entry *ResumeEducation, field string) {
	if field == "" {
		return
	}
	if strings.Contains(field, " - ") {
		left, right, _ := strings.Cut(field, " - ")
		applyEducationField(entry, strings.TrimSpace(left))
		applyEducationField(entry, strings.TrimSpace(right))
		return
	}
	if start, end := splitDateRange(field); start != "" || end != "" {
		entry.EndDate = strings.TrimSpace(strings.Join([]string{start, end}, " -- "))
		return
	}
	if end := extractEndDate(field); end != "" {
		entry.EndDate = end
		return
	}
	if firstYear(field) != 0 {
		entry.EndDate = field
		return
	}
	if educationDegreeLike(field) {
		if entry.Degree == "" || len(field) > len(entry.Degree) {
			entry.Degree = field
		}
		return
	}
	if educationInstitutionLike(field) {
		if entry.Organization == "" {
			entry.Organization = field
		}
		return
	}
	if entry.Location == "" && educationLocationLike(field) && !strings.EqualFold(field, entry.Organization) && !strings.EqualFold(field, entry.Degree) {
		entry.Location = field
	}
}

func educationDegreeLike(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "bachelor") || strings.Contains(lower, "master") || strings.Contains(lower, "phd") || strings.Contains(lower, "diploma") || strings.Contains(lower, "certificate") || strings.Contains(lower, "doctor") || strings.Contains(lower, "engineering") || strings.Contains(lower, "technology") || strings.Contains(lower, "science") || strings.Contains(lower, "information")
}

func educationInstitutionLike(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "university") || strings.Contains(lower, "college") || strings.Contains(lower, "institute") || strings.Contains(lower, "vit")
}

func educationLocationLike(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || educationDegreeLike(value) || firstYear(value) != 0 {
		return false
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "remote") || strings.Contains(lower, "online") {
		return true
	}
	if strings.Contains(value, ",") {
		return true
	}
	words := strings.Fields(value)
	if len(words) < 2 || len(words) > 4 {
		return false
	}
	for _, word := range words {
		trimmed := strings.Trim(word, ".,()")
		if trimmed == "" {
			return false
		}
		first := rune(trimmed[0])
		if first < 'A' || first > 'Z' {
			return false
		}
	}
	return !educationInstitutionLike(value)
}

func extractEducationLocation(content string) string {
	for _, line := range nonEmptyLines(content) {
		for _, field := range educationFields(line) {
			field = strings.TrimSpace(strings.TrimPrefix(field, "-"))
			if educationLocationLike(field) {
				return field
			}
		}
	}
	return ""
}

func educationFields(line string) []string {
	line = strings.NewReplacer("•", "|", "·", "|", "–", "--", "—", "--").Replace(line)
	fields := []string{}
	for _, field := range strings.Split(line, "|") {
		field = strings.TrimSpace(field)
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

func linkLabelFromURL(rawURL string) string {
	label := strings.TrimPrefix(strings.TrimPrefix(rawURL, "https://"), "http://")
	label = strings.TrimPrefix(label, "www.")
	if host, _, ok := strings.Cut(label, "/"); ok {
		label = host
	}
	if label == "" {
		return "Link"
	}
	return label
}

func extractEndDate(text string) string {
	text = strings.TrimSpace(text)
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '-' || r == 0x2013 || r == 0x2014
	})
	if len(parts) >= 2 {
		date := strings.TrimSpace(parts[len(parts)-1])
		if strings.Contains(date, "/") || strings.Contains(date, "Present") || strings.Contains(date, "present") {
			return date
		}
	}
	return ""
}

type resumeTexPromptInput struct {
	Headline    string
	Summary     string
	ContactLine string
	SkillsLine  string
	Skills      []ResumeSkill
	Experience  []ResumeEntry
	Projects    []ResumeEntry
	Education   []ResumeEducation
	Sections    []SourceSection
	Facts       []EvidenceFact
	Claims      []CandidateClaim
	Drafts      []TailoredBulletDraft
	Analysis    JobAnalysis
	Strategy    ApplicationStrategy
}

func buildResumeTexPrompt(input resumeTexPromptInput) string {
	profileJSON, _ := jsonMarshalNoErr(map[string]any{
		"headline":     input.Headline,
		"summary":      input.Summary,
		"contact_line": input.ContactLine,
		"skills_line":  input.SkillsLine,
		"skills":       input.Skills,
		"experience":   input.Experience,
		"projects":     input.Projects,
		"education":    input.Education,
	})
	sourceSectionJSON, _ := jsonMarshalNoErr(sourceSectionSummaries(input.Sections))
	factJSON, _ := jsonMarshalNoErr(factSummaries(input.Facts))
	claimJSON, _ := jsonMarshalNoErr(claimSummaries(input.Claims))
	draftJSON, _ := jsonMarshalNoErr(draftSummaries(input.Drafts))
	jdJSON, _ := jsonMarshalNoErr(map[string]any{
		"role_title":           input.Analysis.RoleTitle,
		"company":              input.Analysis.Company,
		"required_skills":      input.Analysis.RequiredSkills,
		"preferred_skills":     input.Analysis.PreferredSkills,
		"responsibilities":     input.Analysis.Responsibilities,
		"top_pain_points":      input.Analysis.TopPainPoints,
		"positioning_strategy": input.Strategy.PositioningStrategy,
		"resume_headline":      input.Strategy.ResumeHeadline,
		"do_not_overclaim":     input.Strategy.DoNotOverclaim,
	})

	skeleton := latexResumeTemplate()

	return fmt.Sprintf(`# Task
Generate a one-page LaTeX resume that exactly matches the provided format skeleton. Replace ALL placeholder content with tailored content from the context data below. Do NOT invent data not present in the context. Use the best-fit content for the target job description.

# Format Skeleton (use this EXACT formatting, replace only content)
`+"```latex\n%s\n```"+`

# Structured Resume Data
`+"```json\n%s\n```"+`

# Source Section Content (original resume sections)
`+"```json\n%s\n```"+`

# Evidence Facts (atom-level extracted facts)
`+"```json\n%s\n```"+`

# Candidate Claims (approved atom-bank profile claims)
`+"```json\n%s\n```"+`

# Tailored Bullet Drafts (JD-specific bullet drafts)
`+"```json\n%s\n```"+`

# Job Description Analysis
`+"```json\n%s\n```"+`

# Instructions
1. Output ONLY valid LaTeX code matching the skeleton format.
2. Use the structured resume data as the baseline, then improve it only with source sections, evidence facts, approved claims, and tailored drafts.
3. Do NOT invent employers, roles, dates, technologies, metrics, links, achievements, or education.
4. Keep the resume to ONE page: 4-5 bullets for the most relevant current/recent role, 1-2 bullets for older roles, 1 bullet per project, and only the strongest 1-2 projects.
5. Preserve real employer/project/education names, locations, dates, URLs, and contact details from the source sections/profile.
6. Use concise, impact-oriented bullets grounded in facts/claims; prefer JD-relevant bullets but include enough verified breadth to avoid a thin one-role resume.
7. Skills must reflect only technologies evidenced in facts/claims/source sections and should be grouped as Languages, Backend, Frontend, Databases/Infra, AI, and Tools when present.
8. Keep the EXACT same LaTeX commands and formatting as the skeleton.
9. Avoid banned phrases: leveraged, spearheaded, empowered, utilized, cutting-edge, game-changer, transformative, synergy, seamless.
10. If a source value contains special LaTeX characters, escape them correctly.

Return ONLY the LaTeX code, no explanations.`, skeleton, profileJSON, sourceSectionJSON, factJSON, claimJSON, draftJSON, jdJSON)
}

func (s *Store) generateResumeTex(ctx context.Context, prompt string) string {
	if prompt == "" {
		return ""
	}
	system := `You are JD Tailor's resume LaTeX generator. Return ONLY valid LaTeX code matching the provided skeleton format. Never include markdown fences or explanations.`
	rules := s.promptRuleDigest("resume", "validation")
	systemWithRules := system
	if rules != "" {
		systemWithRules = system + "\n\n# Validation Rules\n" + rules
	}
	text, err := s.GenerateLLMText(ctx, nil, systemWithRules, prompt, 2400)
	if err != nil {
		_ = s.LogEvent("warning", "resume tex generation failed: "+err.Error())
		return ""
	}
	text = strings.TrimSpace(text)
	return text
}

func jsonMarshalNoErr(v any) ([]byte, error) {
	return json.Marshal(v)
}

func sourceSectionSummaries(sections []SourceSection) []map[string]any {
	result := []map[string]any{}
	for _, section := range sections {
		result = append(result, map[string]any{
			"heading":      section.Heading,
			"section_type": section.SectionType,
			"content":      section.Content,
		})
	}
	return result
}

func factSummaries(facts []EvidenceFact) []map[string]any {
	result := []map[string]any{}
	for _, fact := range facts {
		result = append(result, map[string]any{
			"fact_text":      fact.FactText,
			"evidence_quote": fact.EvidenceQuote,
			"technologies":   fact.Technologies,
			"origin_heading": fact.OriginHeading,
			"origin_type":    fact.OriginType,
		})
	}
	return result
}

func claimSummaries(claims []CandidateClaim) []map[string]any {
	result := []map[string]any{}
	for _, claim := range claims {
		if claim.Status != "approved" && claim.Status != "approved_restricted" {
			continue
		}
		result = append(result, map[string]any{
			"claim_text":     claim.ClaimText,
			"technologies":   claim.Technologies,
			"actions":        claim.Actions,
			"capabilities":   claim.Capabilities,
			"origin_heading": claim.OriginHeading,
			"origin_type":    claim.OriginType,
			"status":         claim.Status,
		})
	}
	return result
}

func draftSummaries(drafts []TailoredBulletDraft) []map[string]any {
	result := []map[string]any{}
	for _, draft := range drafts {
		result = append(result, map[string]any{
			"draft_text":     draft.DraftText,
			"origin_heading": draft.OriginHeading,
			"origin_type":    draft.OriginType,
		})
	}
	return result
}

// extractDatesFromSection extracts start and end dates from a source section's content
func extractDatesFromSection(section SourceSection) (startDate, endDate string) {
	lines := strings.Split(section.Content, "\n")
	if len(lines) < 2 {
		return "", ""
	}
	// First line typically has: Company | Location | Role
	// Second line typically has: dates
	dateLine := strings.TrimSpace(lines[len(lines)-1])
	parts := strings.Split(dateLine, "--")
	if len(parts) >= 2 {
		startDate = strings.TrimSpace(parts[0])
		endDate = strings.TrimSpace(parts[1])
	}
	return startDate, endDate
}

func parseSectionMetadata(section SourceSection) (company, location, title, startDate, endDate string) {
	lines := nonEmptyLines(section.Content)
	if len(lines) == 0 {
		return section.Heading, "", "", "", ""
	}

	parts := strings.Split(strings.TrimSpace(lines[0]), "|")
	if len(parts) >= 1 {
		company = strings.TrimSpace(stripURLs(parts[0]))
	}
	for _, raw := range parts[1:] {
		part := strings.TrimSpace(stripURLs(raw))
		if part == "" || resumeLinkLabelLike(part) {
			continue
		}
		if location == "" && resumeLocationLike(part) {
			location = part
		}
	}

	if len(lines) >= 2 {
		secondParts := strings.Split(strings.TrimSpace(lines[1]), "|")
		if len(secondParts) >= 1 {
			title = strings.TrimSpace(secondParts[0])
		}
		if len(secondParts) >= 2 {
			datePart := strings.TrimSpace(secondParts[len(secondParts)-1])
			dateParts := strings.Split(datePart, "--")
			if len(dateParts) >= 1 {
				startDate = strings.TrimSpace(dateParts[0])
			}
			if len(dateParts) >= 2 {
				endDate = strings.TrimSpace(dateParts[1])
			}
		}
	}
	return company, location, title, startDate, endDate
}

func extractCompanyURL(section SourceSection) string {
	text := section.Heading + "\n" + section.Content
	url := firstURLInText(text)
	if url == "" {
		return ""
	}
	return "$|$ \\href{" + url + "}{" + sanitizeLaTeX(linkLabelNearURL(text, url)) + "}"
}

func firstURLInText(text string) string {
	for _, field := range strings.Fields(text) {
		field = strings.Trim(field, "<>[](){}.,;'")
		if strings.HasPrefix(field, "https://") || strings.HasPrefix(field, "http://") {
			return field
		}
	}
	return ""
}

func stripURLs(text string) string {
	fields := strings.Fields(text)
	kept := []string{}
	for _, field := range fields {
		trimmed := strings.Trim(field, "<>[](){}.,;'")
		if strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "http://") {
			continue
		}
		kept = append(kept, field)
	}
	return strings.TrimSpace(strings.Join(kept, " "))
}

func linkLabelNearURL(text string, url string) string {
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == '|' || r == '\n' || r == '\t' })
	for _, part := range parts {
		if !strings.Contains(part, url) {
			continue
		}
		label := strings.TrimSpace(stripURLs(part))
		if resumeLinkLabelLike(label) {
			return label
		}
	}
	return linkLabelFromURL(url)
}

func resumeLinkLabelLike(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "website", "site", "github", "gitlab", "portfolio", "demo", "link", "repo", "repository":
		return true
	default:
		return false
	}
}

func resumeLocationLike(value string) bool {
	if educationLocationLike(value) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "remote" || lower == "hybrid" || lower == "onsite" || strings.Contains(lower, "remote")
}

func latexResumeTemplate() string {
	return `%-------------------------
% Resume in Latex
% Author : [[tex .Headline]]
%------------------------

\documentclass[letterpaper,11pt]{article}
\usepackage{lmodern}
\usepackage{latexsym}
\usepackage[empty]{fullpage}
\usepackage{titlesec}
\usepackage[usenames,dvipsnames]{color}
\usepackage{enumitem}
\usepackage[hidelinks]{hyperref}
\usepackage{fancyhdr}
\usepackage[english]{babel}
\usepackage{tabularx}

\hypersetup{
    colorlinks=true,
    urlcolor=blue
}

\pagestyle{fancy}
\fancyhf{}
\fancyfoot{}
\renewcommand{\headrulewidth}{0pt}
\renewcommand{\footrulewidth}{0pt}

%----------PAGE SETUP----------
\addtolength{\oddsidemargin}{-0.5in}
\addtolength{\evensidemargin}{-0.5in}
\addtolength{\textwidth}{1in}
\addtolength{\topmargin}{-.62in}
\addtolength{\textheight}{1.22in}

\urlstyle{same}
\raggedbottom
\raggedright
\setlength{\tabcolsep}{0in}

%----------SECTION FORMAT----------
\titleformat{\section}{
  \vspace{-3pt}\scshape\raggedright\large
}{}{0em}{}[\color{black}\titlerule \vspace{-4pt}]

%----------CUSTOM COMMANDS----------
\newcommand{\resumeItem}[1]{
  \item\small{{#1 \vspace{-3pt}}}
}

\newcommand{\resumeSubheading}[4]{
  \vspace{-1pt}\item
    \begin{tabular*}{0.97\textwidth}[t]{l@{\extracolsep{\fill}}r}
      \textbf{#1} & #2 \\
      \textit{\small#3} & \textit{\small #4} \\
    \end{tabular*}\vspace{-7pt}
}

\newcommand{\resumeProjectHeading}[2]{
    \item
    \begin{tabular*}{0.97\textwidth}{l@{\extracolsep{\fill}}r}
      \small#1 & #2 \\
    \end{tabular*}\vspace{-7pt}
}

\renewcommand\labelitemii{$\vcenter{\hbox{\tiny$\bullet$}}$}

\newcommand{\resumeSubHeadingListStart}{\begin{itemize}[leftmargin=0.15in, label={}]}
\newcommand{\resumeSubHeadingListEnd}{\end{itemize}}
\newcommand{\resumeItemListStart}{\begin{itemize}[leftmargin=0.18in]}
\newcommand{\resumeItemListEnd}{\end{itemize}\vspace{-6pt}}

\begin{document}

%----------HEADING----------
\begin{center}
    \textbf{\Huge \scshape [[tex .Headline]]} \\ \vspace{1pt}
    \small
    [[.ContactLine]]
\end{center}

%-----------SUMMARY-----------
\section{Professional Summary}
\small{[[tex .Summary]]} \vspace{-4pt}

%-----------TECHNICAL SKILLS-----------
\section{Technical Skills}
\begin{itemize}[leftmargin=0.15in, label={}]
\small{\item{
[[range .Skills]]\textbf{[[tex .Category]]}{: [[tex (join .Items ", ")]]} \\
[[end]]
}}
\end{itemize}

%-----------EXPERIENCE-----------
\section{Experience}
\resumeSubHeadingListStart
[[range .Experience]]
\resumeSubheading{[[tex .Company]] [[.URL]]}{[[tex .Location]]}{[[tex .Title]]}{[[tex .StartDate]] -- [[tex .EndDate]]}
\resumeItemListStart
[[range .Bullets]]\resumeItem{[[tex .]]}
[[end]]
\resumeItemListEnd
[[end]]
\resumeSubHeadingListEnd

%-----------PROJECTS-----------
\section{Projects}
\resumeSubHeadingListStart
[[range .Projects]]
\resumeProjectHeading{\textbf{[[tex .Title]]}[[.URL]]}{[[tex .StartDate]] -- [[tex .EndDate]]}
\resumeItemListStart
[[range .Bullets]]\resumeItem{[[tex .]]}
[[end]]
\resumeItemListEnd
[[end]]
\resumeSubHeadingListEnd

%-----------EDUCATION-----------
\section{Education}
\resumeSubHeadingListStart
[[range .Education]]
\resumeSubheading{[[tex .Organization]]}{[[tex .Location]]}{[[tex .Degree]]}{[[tex .EndDate]]}
\vspace{3pt}
[[end]]
\resumeSubHeadingListEnd

\end{document}
`
}

func sanitizeLaTeX(input string) string {
	return strings.NewReplacer(
		`\`, `\textbackslash{}`,
		`&`, `\&`,
		`%`, `\%`,
		`$`, `\$`,
		`#`, `\#`,
		`{`, `\{`,
		`}`, `\}`,
		`~`, `\textasciitilde{}`,
		`^`, `\textasciicircum{}`,
	).Replace(input)
}
