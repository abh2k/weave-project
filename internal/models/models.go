package models

type GithubResponse struct {
	TotalCount int     `json:"total_count"`
	Items      []Items `json:"items"`
}

type Items struct {
	FileUrl string     `json:"html_url"`
	Repo    Repository `json:"repository"`
}

type Repository struct {
	RepoUrl  string `json:"html_url"`
	RepoName string `json:"name"`
}
