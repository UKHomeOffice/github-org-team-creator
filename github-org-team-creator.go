package main

import (
	"os"
	"github.com/urfave/cli"
	"fmt"
	"bufio"
	"log"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"strconv"
)

func main() {
	app := cli.NewApp()
	app.Name = "boom"
	app.Usage = "Adds all users in an organisation to a team of your choice"
	app.ArgsUsage = "[Organisation] [Team Name]"
	app.UsageText = fmt.Sprintf("github-org-team-creator %s", app.ArgsUsage)
	app.Action = func(c *cli.Context) error {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Please enter your Github personal access token: ")
		accessToken, _ := reader.ReadString('\n')
		org := c.Args().Get(0)
		team := c.Args().Get(1)
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		githubClient := github.NewClient(tc)
		teamId := getTeamId(org, team, githubClient)
		if teamId == 0 {
			fmt.Println("Creating team...")
			teamId = createTeam(org, team, githubClient)
		}
		fmt.Println("Team has ID of " + strconv.Itoa(teamId))
		fmt.Println("Adding org users to team")
		listAndAddOrgUsersToTeam(org, teamId, githubClient)
		return nil
	}

	app.Run(os.Args)
}

func createTeam(org string, teamName string, githubClient *github.Client) int {
	newTeam := github.Team{Name: &teamName}
	team, _, err := githubClient.Organizations.CreateTeam(org, &newTeam)
	handleError(err)
	return *team.ID
}

func listAndAddOrgUsersToTeam(org string, teamId int, githubClient *github.Client) {
	fmt.Println("Adding page 1 of users to org")
	resp := listAndAddOrgUsersForPage(org, teamId, 1, githubClient)
	if resp.LastPage > 1 {
		for i := 2; i <= resp.LastPage; i++ {
			fmt.Println("Adding page " + strconv.Itoa(i) + " of users to org")
			listAndAddOrgUsersForPage(org, teamId, i, githubClient)
		}
	}
}

func listAndAddOrgUsersForPage(org string, teamId int, page int, githubClient *github.Client) *github.Response {
	ListMembersOpt := &github.ListMembersOptions{
		PublicOnly:  false,
		Filter:      "2fa_enabled",
		ListOptions: github.ListOptions{PerPage: 100, Page: page},
	}
	var users, resp, err = githubClient.Organizations.ListMembers(org, ListMembersOpt)
	handleError(err)
	addOrgUsers(users, teamId, githubClient)
	return resp
}

func addOrgUsers(users []*github.User, teamId int, githubClient *github.Client) {
	addTeamMemberOpt := &github.OrganizationAddTeamMembershipOptions{Role: "member"}
	for _, user := range users {
		userLogin := *user.Login
		githubClient.Organizations.AddTeamMembership(teamId, userLogin, addTeamMemberOpt)
		fmt.Println(userLogin + " added to team")
	}
}

func getTeamId(org string, team string, githubClient *github.Client) int {
	//	TODO: Get this to support paging
	var members, _, err = githubClient.Organizations.ListTeams(org, &github.ListOptions{PerPage: 1000})
	handleError(err)
	for _, mem := range members {
		if *mem.Name == team {
			fmt.Println("Github team " + team + " already exists")
			return *mem.ID
		}
	}
	fmt.Println("Github team " + team + " doesn't exist")
	return 0
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
