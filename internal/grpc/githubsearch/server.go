package githubsearch

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	apiv1 "weave-project/gen/proto/weave/api/v1"
	"weave-project/internal/constants"
	"weave-project/internal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	apiv1.UnimplementedGithubSearchServiceServer
	httpClient *http.Client
	logger     *log.Logger
	gitToken   string
}

func NewServer(gitToken string) *Server {
	return &Server{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:   log.Default(),
		gitToken: gitToken,
	}
}

func (s *Server) Search(ctx context.Context, request *apiv1.SearchRequest) (*apiv1.SearchResponse, error) {
	if err := validate(request); err != nil {
		s.logger.Printf("request validation failed: %s", err.Error())
		return emptyResponse(), err
	}

	searchURL, err := buildSearchURL(request.SearchTerm, request.User)
	if err != nil {
		s.logger.Printf("failed to build github search URL: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Internal, "failed to build github search request")
	}

	req, err := s.newGitHubRequest(ctx, searchURL)
	if err != nil {
		s.logger.Printf("failed to create github request: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Internal, "failed to create github request")
	}

	s.logger.Printf("raw URL: %s", req.URL)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Printf("github request failed: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Internal, "github service is unavailable")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Printf("failed to read github response body: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Internal, "failed to read github response")
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Printf("github api error. status=%d body=%s", resp.StatusCode, string(body))
		err := processStatusCode(resp.StatusCode)
		return emptyResponse(), err
	}

	response, err := createSearchResponse(body)
	if err != nil {
		s.logger.Printf("error occurred while creating search response: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Internal, "failed to parse github response")
	}

	return response, nil
}

func validate(request *apiv1.SearchRequest) error {
	if request == nil {
		return status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}
	if strings.TrimSpace(request.SearchTerm) == "" {
		return status.Errorf(codes.InvalidArgument, "search term cannot be empty")
	}
	return nil
}

func buildGithubQuery(searchTerm string, user string) string {
	query := strings.TrimSpace(searchTerm)
	if strings.TrimSpace(user) != "" {
		query = searchTerm + " user:" + strings.TrimSpace(user)
	}
	return query
}

func buildSearchURL(searchTerm string, user string) (*url.URL, error) {
	searchURL, err := url.Parse(constants.GITHUB_API + constants.SEARCH_CODE_ENDPOINT)
	if err != nil {
		return nil, err
	}

	query := searchURL.Query()
	query.Set("q", buildGithubQuery(searchTerm, user))
	searchURL.RawQuery = query.Encode()
	return searchURL, nil
}

func (s *Server) newGitHubRequest(ctx context.Context, searchURL *url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+s.gitToken)
	return req, nil
}

func createSearchResponse(body []byte) (*apiv1.SearchResponse, error) {
	var respObject models.GithubResponse
	if err := json.Unmarshal(body, &respObject); err != nil {
		return nil, err
	}

	response := &apiv1.SearchResponse{}
	for _, item := range respObject.Items {
		searchItem := apiv1.Result{
			FileUrl: item.FileUrl,
			Repo:    item.Repo.FullName,
		}
		response.Results = append(response.Results, &searchItem)
	}

	return response, nil
}

func emptyResponse() *apiv1.SearchResponse {
	return &apiv1.SearchResponse{Results: []*apiv1.Result{}}
}

func processStatusCode(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return status.Errorf(codes.Unauthenticated, "invalid github token")
	case http.StatusForbidden:
		return status.Errorf(codes.ResourceExhausted, "github rate limited")
	default:
		return status.Errorf(codes.Unavailable, "github raised non-success code : %d", statusCode)
	}
}
