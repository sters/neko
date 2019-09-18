package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sters/neko/goauth2"
	"github.com/sters/neko/gphoto"
)

type config struct {
	ClientID     string `envconfig:"GOOGLE_CLIENT_ID" required:"true"`
	ClientSecret string `envconfig:"GOOGLE_CLIENT_SECRET" required:"true"`
	RefreshToken string `envconfig:"GOOGLE_REFRESH_TOKEN"`
}

func main() {
	var err error
	var cfg config
	err = envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	oauth2 := goauth2.NewClient(
		strings.TrimSpace(cfg.ClientID),
		strings.TrimSpace(cfg.ClientSecret),
	)
	oauth2.WithScopes(gphoto.ScopeLibraryReadOnly)
	oauth2.WithHTTPClient(&http.Client{
		Timeout: 5 * time.Second,
	})

	if cfg.RefreshToken == "" {
		for {
			fmt.Println("Open in your web browser:")
			fmt.Printf("%s\n", oauth2.GetOAuthURI())

			fmt.Println("")
			fmt.Print("Input authorization code > ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()

			fmt.Println("Authorizing...")
			err = oauth2.Authorization(
				context.Background(),
				strings.TrimSpace(scanner.Text()),
			)
			if err != nil {
				log.Fatalf("%+v", err)
			}

			if oauth2.GetRefreshToken() == "" {
				fmt.Println("\nSomething wrong, retry authorization.\n")
				continue
			}

			break
		}

		cfg.RefreshToken = oauth2.GetRefreshToken()
	}

	fmt.Println("Refreshing AccessToken...")
	err = oauth2.Refresh(
		context.Background(),
		strings.TrimSpace(cfg.RefreshToken),
	)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	fmt.Println("Authorized!")
	log.Println("AccessToken : " + oauth2.GetAccessToken())
	log.Println("RefreshToken: " + oauth2.GetRefreshToken())

	gphotoClient := gphoto.NewClient(
		&http.Client{
			Timeout: 5 * time.Second,
		},
		oauth2.GetAccessToken(),
	)

	resp, err := gphotoClient.MediaItemsSearch(
		context.Background(),
		&gphoto.MediaItemsSearchRequest{
			PagerRequest: gphoto.PagerRequest{
				PageSize: "100",
			},
			Filters: &gphoto.Filters{
				ContentFilter: &gphoto.ContentFilter{
					IncludedContentCategories: []gphoto.ContentCategory{
						gphoto.ContentCategoryPets,
					},
				},
			},
		},
	)

	// collected pet images!
	for _, item := range resp.MediaItems {
		log.Print(item.ProductURL)
	}
}
