package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// --- Templates ---
// Notice: We use {{ .ResourceName }} which is calculated in Go
const membershipTemplate = `---
apiVersion: user.github.upbound.io/v1alpha1
kind: Membership
metadata:
  labels:
    suse.okta.com/user-id: '{{ .OktaID }}'
  name: {{ .ResourceName }}
spec:
  deletionPolicy: Delete
  forProvider:
    downgradeOnDestroy: false
    role: {{ .Role }}
    username: {{ .GithubUsername }}
  providerConfigRef:
    name: {{ .GithubOrg }}`

const teamMembershipTemplate = `---
apiVersion: team.github.upbound.io/v1alpha1
kind: TeamMembership
metadata:
  name: {{ .ResourceName }}--{{ .TeamSlug }}
spec:
  forProvider:
    org: {{ .GithubOrg }}
    teamSlug: {{ .TeamSlug }}
    username: {{ .GithubUsername }}
    role: {{ .TeamRole }}
  providerConfigRef:
    name: {{ .GithubOrg }}`

// --- Data Structures ---

type User struct {
	GithubUsername string
	OktaID         string
	OrgRole        string
}

type Config struct {
	GithubOrg string
	Users     []User
	Teams     []string
	TeamRole  string
}

// --- Main ---

func main() {
	var config Config
	var users []User

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1).
		SetString("🐙 Octocrew: GitOps Onboarding")

	fmt.Fprintln(os.Stderr, headerStyle)

	// 1. Global Config (Org)
	formOrg := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Target Organization").
				Placeholder("suse").
				Value(&config.GithubOrg).
				Validate(isNotEmpty),
		),
	).WithProgramOptions(tea.WithOutput(os.Stderr))
	runForm(formOrg)

	// 2. User Loop
	for {
		var u User
		var addAnother bool

		formUser := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("GitHub Username").
					Value(&u.GithubUsername).
					Validate(isNotEmpty),
				huh.NewInput().
					Title("Okta ID").
					Value(&u.OktaID).
					Validate(isNotEmpty),
				huh.NewSelect[string]().
					Title("Org Role").
					Options(
						huh.NewOption("Member", "member"),
						huh.NewOption("Admin", "admin"),
					).
					Value(&u.OrgRole),
				huh.NewConfirm().
					Title("Add another user?").
					Value(&addAnother),
			),
		).WithProgramOptions(tea.WithOutput(os.Stderr))
		runForm(formUser)

		users = append(users, u)
		if !addAnother {
			break
		}
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("---"))
	}
	config.Users = users

	// 3. Team Chaining
	var addToTeams bool
	var teamsStr string

	formConfirmTeam := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add these users to Teams?").
				Value(&addToTeams),
		),
	).WithProgramOptions(tea.WithOutput(os.Stderr))
	runForm(formConfirmTeam)

	if addToTeams {
		formTeamInput := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Teams (comma separated)").
					Placeholder("engineering, devops").
					Value(&teamsStr).
					Validate(isNotEmpty),
				huh.NewSelect[string]().
					Title("Team Role").
					Options(
						huh.NewOption("Member", "member"),
						huh.NewOption("Maintainer", "maintainer"),
					).
					Value(&config.TeamRole),
			),
		).WithProgramOptions(tea.WithOutput(os.Stderr))
		runForm(formTeamInput)

		parts := strings.Split(teamsStr, ",")
		for _, p := range parts {
			clean := strings.TrimSpace(p)
			if clean != "" {
				config.Teams = append(config.Teams, clean)
			}
		}
	}

	// 4. Generation
	generateOutput(config)
}

func runForm(f *huh.Form) {
	err := f.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func isNotEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("cannot be empty")
	}
	return nil
}

// makeResourceName handles the specific logic for naming
func makeResourceName(org, username string) string {
	org = strings.TrimSpace(org)
	username = strings.TrimSpace(username)

	return fmt.Sprintf("%s--%s", org, username)
}

func generateOutput(c Config) {
	tmplMember, _ := template.New("member").Parse(membershipTemplate)
	tmplTeam, _ := template.New("team").Parse(teamMembershipTemplate)

	outDir := c.GithubOrg
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "Error creating directory:", err)
		os.Exit(1)
	}

	for _, u := range c.Users {
		var sb strings.Builder
		resourceName := makeResourceName(c.GithubOrg, u.GithubUsername)

		data := map[string]string{
			"OktaID":         u.OktaID,
			"GithubOrg":      c.GithubOrg,
			"GithubUsername": u.GithubUsername,
			"ResourceName":   resourceName,
			"Role":           u.OrgRole,
		}
		tmplMember.Execute(&sb, data)

		for _, team := range c.Teams {
			sb.WriteString("\n")
			teamData := map[string]string{
				"GithubOrg":      c.GithubOrg,
				"GithubUsername": u.GithubUsername,
				"TeamSlug":       team,
				"ResourceName":   resourceName,
				"TeamRole":       c.TeamRole,
			}
			tmplTeam.Execute(&sb, teamData)
		}

		filePath := filepath.Join(outDir, u.GithubUsername+".yaml")
		if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
			fmt.Fprintln(os.Stderr, "Error writing file:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  wrote "+filePath))
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	fmt.Fprintln(os.Stderr, successStyle.Render(fmt.Sprintf("\nDone! Generated %d resources in ./%s/", len(c.Users)+(len(c.Users)*len(c.Teams)), outDir)))
}
