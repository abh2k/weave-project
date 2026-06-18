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

func NewServer(git_token string) *Server {
	return &Server{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:   log.Default(),
		gitToken: git_token,
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
		return emptyResponse(), status.Errorf(codes.Unavailable, "github service is unavailable")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Printf("failed to read github response body: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Unavailable, "failed to read github response")
	}

	if err := checkGitHubResponse(resp.StatusCode, body); err != nil {
		s.logger.Printf("github api error. status=%d body=%s", resp.StatusCode, string(body))
		return emptyResponse(), err
	}

	respObject, err := decodeGitHubResponse(body)
	if err != nil {
		s.logger.Printf("error occurred while unmarshalling response body: %s", err.Error())
		return emptyResponse(), status.Errorf(codes.Internal, "failed to parse github response")
	}

	response := &apiv1.SearchResponse{}
	for _, item := range respObject.Items {
		search_item := apiv1.Result{
			FileUrl: item.FileUrl,
			Repo:    item.Repo.RepoName,
		}
		response.Results = append(response.Results, &search_item)
	}

	return response, nil
}

func validate(request *apiv1.SearchRequest) error {
	if request == nil {
		return status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}
	if request.SearchTerm == "" {
		return status.Errorf(codes.InvalidArgument, "search term cannot be empty")
	}
	if strings.TrimSpace(request.SearchTerm) == "" {
		return status.Errorf(codes.InvalidArgument, "search term cannot be whitespace")
	}
	return nil
}

func buildGithubQuery(searchTerm string, user string) string {
	query := searchTerm
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

func checkGitHubResponse(statusCode int, body []byte) error {
	if statusCode == http.StatusOK {
		return nil
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return status.Errorf(codes.Unauthenticated, "github authentication failed")
	case http.StatusForbidden:
		return status.Errorf(codes.PermissionDenied, "github request forbidden")
	case http.StatusUnprocessableEntity:
		return status.Errorf(codes.InvalidArgument, "invalid github search query")
	case http.StatusTooManyRequests:
		return status.Errorf(codes.ResourceExhausted, "github rate limit exceeded")
	default:
		if statusCode >= http.StatusInternalServerError {
			return status.Errorf(codes.Unavailable, "github service is unavailable")
		}
		return status.Errorf(codes.Unknown, "github api call failed with status %d: %s", statusCode, string(body))
	}
}

func decodeGitHubResponse(body []byte) (*models.GithubResponse, error) {
	var respObject models.GithubResponse
	if err := json.Unmarshal(body, &respObject); err != nil {
		return nil, err
	}
	return &respObject, nil
}

func emptyResponse() *apiv1.SearchResponse {
	return &apiv1.SearchResponse{Results: []*apiv1.Result{}}
}
