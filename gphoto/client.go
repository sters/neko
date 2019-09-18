package gphoto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/morikuni/failure"
	"github.com/sters/neko/gclient"
)

const (
	baseURL                  = "https://photoslibrary.googleapis.com/v1/"
	mediaItemsSearchEndpoint = "mediaItems:search"

	ScopeLibraryReadOnly = "https://www.googleapis.com/auth/photoslibrary.readonly"
)

type (
	Client struct {
		c     *http.Client
		token string
	}

	PagerRequest struct {
		PageSize  string `json:"pageSize,omitempty"`
		PageToken string `json:"pageToken,omitempty"`
	}
	PagerResponse struct {
		NextPageToken string `json:"nextPageToken,omitempty"`
	}

	// MediaItemsSearchRequest see https://developers.google.com/photos/library/reference/rest/v1/mediaItems/search
	MediaItemsSearchRequest struct {
		PagerRequest
		AlbumID string   `json:"albumId,omitempty"`
		Filters *Filters `json:"filters,omitempty"`
	}

	Filters struct {
		DateFilter               *DateFilter      `json:"dateFilter,omitempty"`
		ContentFilter            *ContentFilter   `json:"contentFilter,omitempty"`
		MediaTypeFilter          *MediaTypeFilter `json:"mediaTypeFilter,omitempty"`
		FeatureFilter            *FeatureFilter   `json:"featureFilter,omitempty"`
		IncludeArchivedMedia     bool             `json:"includeArchivedMedia,omitempty"`
		ExcludeNonAppCreatedData bool             `json:"excludeNonAppCreatedData,omitempty"`
	}

	DateFilter struct {
		Dates  []*Date      `json:"dates,omitempty"`
		Ranges []*DateRange `json:"ranges,omitempty"`
	}
	Date struct {
		Year  int `json:"year,omitempty"`
		Month int `json:"month,omitempty"`
		Day   int `json:"day,omitempty"`
	}
	DateRange struct {
		StartDate *Date `json:"startDate,omitempty"`
		EndDate   *Date `json:"endDate,omitempty"`
	}

	ContentFilter struct {
		IncludedContentCategories []ContentCategory `json:"includedContentCategories,omitempty"`
		ExcludedContentCategories []ContentCategory `json:"excludedContentCategories,omitempty"`
	}
	ContentCategory string

	MediaTypeFilter struct {
		MediaTypes []MediaType `json:"mediaTypes,omitempty"`
	}
	MediaType string

	FeatureFilter struct {
		IncludedFeatures []Feature `json:"includedFeatures,omitempty"`
	}
	Feature string

	MediaItemsSearchResponse struct {
		PagerResponse
		MediaItems []*MediaItem `json:"mediaItems,omitempty"`
	}

	MediaItem struct {
		ID              string           `json:"id,omitempty"`
		Description     string           `json:"description,omitempty"`
		ProductURL      string           `json:"productUrl,omitempty"`
		BaseURL         string           `json:"baseUrl,omitempty"`
		MimeType        string           `json:"mimeType,omitempty"`
		MediaMetadata   *MediaMetadata   `json:"mediaMetadata,omitempty"`
		ContributorInfo *ContributorInfo `json:"contributorInfo,omitempty"`
		Filename        string           `json:"filename,omitempty"`
	}
	MediaMetadata struct {
		CreationTime string `json:"creationTime,omitempty"`
		Width        string `json:"width,omitempty"`
		Height       string `json:"height,omitempty"`
		Photo        *Photo `json:"photo,omitempty"`
		Video        *Video `json:"video,omitempty"`
	}
	Photo struct {
		CameraMake      string  `json:"cameraMake,omitempty"`
		CameraModel     string  `json:"cameraModel,omitempty"`
		FocalLength     float64 `json:"focalLength,omitempty"`
		ApertureFNumber float64 `json:"apertureFNumber,omitempty"`
		IsoEquivalent   float64 `json:"isoEquivalent,omitempty"`
		ExposureTime    string  `json:"exposureTime,omitempty"`
	}
	Video struct {
		CameraMake  string                `json:"cameraMake,omitempty"`
		CameraModel string                `json:"cameraModel,omitempty"`
		Fps         float64               `json:"fps,omitempty"`
		Status      VideoProcessingStatus `json:"status,omitempty"`
	}
	VideoProcessingStatus string

	ContributorInfo struct {
		ProfilePictureBaseURL string `json:"profilePictureBaseUrl,omitempty"`
		DisplayName           string `json:"displayName,omitempty"`
	}
)

const (
	ContentCategoryNone         ContentCategory = "NONE"
	ContentCategoryLandscapes   ContentCategory = "LANDSCAPES"
	ContentCategoryReceipts     ContentCategory = "RECEIPTS"
	ContentCategoryCityscapes   ContentCategory = "CITYSCAPES"
	ContentCategoryLandmarks    ContentCategory = "LANDMARKS"
	ContentCategorySelfies      ContentCategory = "SELFIES"
	ContentCategoryPeople       ContentCategory = "PEOPLE"
	ContentCategoryPets         ContentCategory = "PETS"
	ContentCategoryWeddings     ContentCategory = "WEDDINGS"
	ContentCategoryBirthdays    ContentCategory = "BIRTHDAYS"
	ContentCategoryDocuments    ContentCategory = "DOCUMENTS"
	ContentCategoryTravel       ContentCategory = "TRAVEL"
	ContentCategoryAnimals      ContentCategory = "ANIMALS"
	ContentCategoryFood         ContentCategory = "FOOD"
	ContentCategorySport        ContentCategory = "SPORT"
	ContentCategoryNight        ContentCategory = "NIGHT"
	ContentCategoryPerformances ContentCategory = "PERFORMANCES"
	ContentCategoryWhiteboards  ContentCategory = "WHITEBOARDS"
	ContentCategoryScreenshots  ContentCategory = "SCREENSHOTS"
	ContentCategoryUtility      ContentCategory = "UTILITY"
	ContentCategoryArts         ContentCategory = "ARTS"
	ContentCategoryCrafts       ContentCategory = "CRAFTS"
	ContentCategoryFashion      ContentCategory = "FASHION"
	ContentCategoryHouses       ContentCategory = "HOUSES"
	ContentCategoryGardens      ContentCategory = "GARDENS"
	ContentCategoryFlowers      ContentCategory = "FLOWERS"
	ContentCategoryHolidays     ContentCategory = "HOLIDAYS"

	MediaTypeAllMedia MediaType = "ALL_MEDIA"
	MediaTypeVideo    MediaType = "VIDEO"
	MediaTypePhoto    MediaType = "PHOTO"

	FeatureNone      Feature = "NONE"
	FeatureFavorites Feature = "FAVORITES"

	VideoProcessingStatusUnspecified VideoProcessingStatus = "UNSPECIFIED"
	VideoProcessingStatusProcessing  VideoProcessingStatus = "PROCESSING"
	VideoProcessingStatusReady       VideoProcessingStatus = "READY"
	VideoProcessingStatusFailed      VideoProcessingStatus = "FAILED"
)

func NewClient(c *http.Client, token string) *Client {
	return &Client{
		c:     c,
		token: token,
	}
}

func (c *Client) request(ctx context.Context, endpoint string, req interface{}, response interface{}) error {
	buf, err := json.Marshal(req)
	if err != nil {
		return failure.Wrap(err)
	}

	rawRequest, err := http.NewRequest(
		http.MethodPost,
		baseURL+endpoint,
		bytes.NewBuffer(buf),
	)
	if err != nil {
		return failure.Wrap(err)
	}

	rawRequest = rawRequest.WithContext(ctx)

	rawRequest.Header = http.Header{}
	rawRequest.Header.Set(
		gclient.AuthorizationHeader,
		fmt.Sprintf(gclient.AuthorizationParam, c.token),
	)
	rawRequest.Header.Set(gclient.ContentTypeHeader, gclient.ContentTypeJSON)

	rawResponse, err := c.c.Do(rawRequest)
	if err != nil {
		return failure.Wrap(err)
	}

	responseBuf, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		return failure.Wrap(err)
	}

	err = json.Unmarshal(responseBuf, response)
	if err != nil {
		return failure.Wrap(err)
	}

	return nil
}

func (c *Client) MediaItemsSearch(ctx context.Context, req *MediaItemsSearchRequest) (*MediaItemsSearchResponse, error) {
	var response MediaItemsSearchResponse
	err := c.request(
		ctx,
		mediaItemsSearchEndpoint,
		req,
		&response,
	)
	return &response, err
}
