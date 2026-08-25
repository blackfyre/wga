package landing

import (
	"fmt"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils/url"
)

const recentArtworkLimit = 4

func buildHomePage(repo *repositories.LandingRepository, date time.Time) (pages.HomePage, error) {
	artistCount, err := repo.CountPublishedArtists()
	if err != nil {
		return pages.HomePage{}, err
	}
	artworkCount, err := repo.CountPublishedArtworks()
	if err != nil {
		return pages.HomePage{}, err
	}
	schoolCount, err := repo.CountSchools()
	if err != nil {
		return pages.HomePage{}, err
	}

	eligibleArtworkCount, err := repo.CountEligibleArtworks()
	if err != nil {
		return pages.HomePage{}, err
	}
	var featured *repositories.LandingArtwork
	if eligibleArtworkCount > 0 {
		index := dailyArtworkIndex(date, eligibleArtworkCount)
		featured, err = repo.FindEligibleArtworkByOffset(index)
		if err != nil {
			return pages.HomePage{}, err
		}
	}
	recent, err := repo.ListRecentEligibleArtworks()
	if err != nil {
		return pages.HomePage{}, err
	}

	page := pages.HomePage{
		ArtistCount:    fmt.Sprintf("%d", artistCount),
		ArtworkCount:   fmt.Sprintf("%d", artworkCount),
		SchoolCount:    fmt.Sprintf("%d", schoolCount),
		RecentArtworks: make([]pages.HomeRecentArtwork, 0, len(recent)),
	}
	if featured != nil {
		page.FeaturedArtwork = featuredArtwork(*featured)
	}
	for _, artwork := range recent {
		page.RecentArtworks = append(page.RecentArtworks, recentArtwork(artwork))
	}

	return page, nil
}

func dailyArtworkIndex(date time.Time, count int) int {
	if count <= 0 {
		return 0
	}

	seconds := date.UTC().Unix()
	day := seconds / int64(24*time.Hour/time.Second)
	if seconds < 0 && seconds%int64(24*time.Hour/time.Second) != 0 {
		day--
	}
	index := int(day % int64(count))
	if index < 0 {
		return index + count
	}
	return index
}

func featuredArtwork(work repositories.LandingArtwork) pages.HomeFeaturedArtwork {
	return pages.HomeFeaturedArtwork{
		Title:  work.Artwork.GetString("title"),
		Artist: work.Artist.GetString("name"),
		Year:   work.Artwork.GetString("year"),
		URL:    artworkURL(work),
		Image:  url.GenerateArtworkImageURL(work.Artwork, url.DeliveryProfileFeature, ""),
	}
}

func recentArtwork(work repositories.LandingArtwork) pages.HomeRecentArtwork {
	return pages.HomeRecentArtwork{
		Title:  work.Artwork.GetString("title"),
		Artist: work.Artist.GetString("name"),
		Year:   work.Artwork.GetString("year"),
		URL:    artworkURL(work),
		Image:  url.GenerateArtworkImageURL(work.Artwork, url.DeliveryProfileCardAndArtistIndex, ""),
	}
}

func artworkURL(work repositories.LandingArtwork) string {
	return url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
		ArtistId:     work.Artist.Id,
		ArtistName:   work.Artist.GetString("name"),
		ArtworkId:    work.Artwork.Id,
		ArtworkTitle: work.Artwork.GetString("title"),
	})
}
