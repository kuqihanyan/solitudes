package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/biezhi/gorm-paginator/pagination"
	"github.com/gofiber/fiber"
	"github.com/gorilla/feeds"

	"github.com/naiba/solitudes"
	"github.com/naiba/solitudes/internal/model"
	"github.com/naiba/solitudes/pkg/translator"
)

func archive(c *fiber.Ctx) {
	var page int64
	page, _ = strconv.ParseInt(c.Params("page"), 10, 64)
	var articles []model.Article
	pg := pagination.Paging(&pagination.Param{
		DB:      solitudes.System.DB.Where("book_refer is NULL"),
		Page:    int(page),
		Limit:   15,
		OrderBy: []string{"created_at DESC"},
	}, &articles)
	for i := 0; i < len(articles); i++ {
		articles[i].RelatedCount(solitudes.System.DB, solitudes.System.Pool, checkPoolSubmit)
	}
	c.Status(http.StatusOK).Render("default/archive", injectSiteData(c, fiber.Map{
		"title":    c.Locals(solitudes.CtxTranslator).(*translator.Translator).T("archive"),
		"what":     "archives",
		"articles": listArticleByYear(articles),
		"page":     pg,
	}))
}

func feedHandler(c *fiber.Ctx) {
	if c.Params("format") == "" {
		c.Status(http.StatusBadRequest).JSON(map[string]interface{}{
			"message":         "please spec a feed format",
			"supportedFormat": []string{"json", "rss", "atom"},
			"feedLink":        "https://" + solitudes.System.Config.Site.Domain + "/feed/:format",
		})
		return
	}
	feed := &feeds.Feed{
		Title:       solitudes.System.Config.Site.SpaceName,
		Link:        &feeds.Link{Href: "https://" + solitudes.System.Config.Site.Domain},
		Description: solitudes.System.Config.Site.SpaceDesc,
		Author:      &feeds.Author{Name: solitudes.System.Config.User.Nickname, Email: solitudes.System.Config.User.Email},
		Updated:     time.Now(),
	}
	var articles []model.Article
	solitudes.System.DB.Order("created_at DESC", true).Limit(20).Find(&articles)
	for i := 0; i < len(articles); i++ {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:   articles[i].Title,
			Link:    &feeds.Link{Href: "https://" + solitudes.System.Config.Site.Domain + "/" + articles[i].Slug + "/v" + strconv.Itoa(int(articles[i].Version))},
			Author:  &feeds.Author{Name: solitudes.System.Config.User.Nickname, Email: solitudes.System.Config.User.Email},
			Content: luteEngine.MarkdownStr(articles[i].GetIndexID(), articles[i].Content),
			Created: articles[i].CreatedAt,
			Updated: articles[i].UpdatedAt,
		})
	}
	switch c.Params("format") {
	case "atom":
		atom, err := feed.ToAtom()
		if err != nil {
			c.Status(http.StatusInternalServerError).Write(err)
			return
		}
		c.Set("Content-Type", "application/xml")
		c.Status(http.StatusOK).Write(atom)
	case "rss":
		rss, err := feed.ToRss()
		if err != nil {
			c.Status(http.StatusInternalServerError).Write(err)
			return
		}
		c.Set("Content-Type", "application/xml")
		c.Status(http.StatusOK).Write(rss)
	case "json":
		json, err := feed.ToJSON()
		if err != nil {
			c.Status(http.StatusInternalServerError).Write(err)
			return
		}
		c.Set("Content-Type", "application/json")
		c.Status(http.StatusOK).Write(json)
	default:
		c.Status(http.StatusOK).Write("Unknown type")
	}
}

func tags(c *fiber.Ctx) {
	var page int64
	page, _ = strconv.ParseInt(c.Params("page"), 10, 64)
	var articles []model.Article
	pg := pagination.Paging(&pagination.Param{
		DB:      solitudes.System.DB.Where("tags @> ARRAY[?]::varchar[]", c.Params("tag")),
		Page:    int(page),
		Limit:   15,
		OrderBy: []string{"created_at DESC"},
	}, &articles)
	for i := 0; i < len(articles); i++ {
		articles[i].RelatedCount(solitudes.System.DB, solitudes.System.Pool, checkPoolSubmit)
	}
	c.Status(http.StatusOK).Render("default/archive", injectSiteData(c, fiber.Map{
		"title":    c.Locals(solitudes.CtxTranslator).(*translator.Translator).T("articles_in", c.Params("tag")),
		"what":     "tags",
		"articles": listArticleByYear(articles),
		"page":     pg,
	}))
}

func listArticleByYear(as []model.Article) [][]model.Article {
	var listed [][]model.Article
	var lastYear int
	var listItem []model.Article
	for i := 0; i < len(as); i++ {
		currentYear := as[i].CreatedAt.Year()
		if currentYear != lastYear {
			if len(listItem) > 0 {
				listed = append(listed, listItem)
				listItem = make([]model.Article, 0)
			}
			lastYear = currentYear
		}
		listItem = append(listItem, as[i])
	}
	if len(listItem) > 0 {
		listed = append(listed, listItem)
	}
	return listed
}
