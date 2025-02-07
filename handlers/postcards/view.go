package postcards

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"

	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func viewPostcard(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	// postCardId := c.QueryParamDefault("p", "nope")
	postCardId := cmp.Or(c.Request.URL.Query().Get("p"), "nope")

	if postCardId == "nope" {
		app.Logger().Error(fmt.Sprintf("Invalid postcard id: %s", postCardId))
		return utils.NotFoundError(c)
	}

	r, err := app.FindRecordById("Postcards", postCardId)

	if err != nil {
		app.Logger().Error("Failed to find postcard", "id", postCardId, "error", err.Error())
		return utils.NotFoundError(c)
	}

	if errs := app.ExpandRecord(r, []string{"image_id"}, nil); len(errs) > 0 {
		app.Logger().Error("Failed to expand record", "id", postCardId, "errors", errs)
		return utils.ServerFaultError(c)
	}

	aw := r.ExpandedOne("image_id")

	content := pages.PostcardView{
		SenderName: r.GetString("sender_name"),
		Message:    r.GetString("message"),
		Image:      url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), ""),
		Title:      aw.GetString("title"),
		Comment:    aw.GetString("comment"),
		Technique:  aw.GetString("technique"),
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Postcard")

	// c.Response.Header().Set("HX-Push-Url", fullUrl)

	var buf bytes.Buffer

	err = pages.PostcardPage(content).Render(ctx, &buf)

	if err != nil {
		app.Logger().Error("Error rendering artwork page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buf.String())
}
