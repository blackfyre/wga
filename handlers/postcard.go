package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/assets/templ/components"
	"github.com/blackfyre/wga/assets/templ/error_pages"
	"github.com/blackfyre/wga/assets/templ/pages"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

// registerPostcardHandlers registers the postcard handlers
func registerPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("postcard/send", func(c echo.Context) error {

			if !utils.IsHtmxRequest(c) {
				return apis.NewBadRequestError("Unexpected request", nil)
			}

			//get awid query param
			awid := c.QueryParam("awid")

			//check if awid is empty
			if awid == "" {
				app.Logger().Error("awid is empty on postcard/send")
				return apis.NewBadRequestError("awid is empty", nil)
			}

			ctx := context.Background()

			r, err := app.Dao().FindRecordById("artworks", awid)

			if err != nil {
				app.Logger().Error("Failed to find artwork "+awid, err)
				return utils.NotFoundError(c)
			}

			err = components.PostcardEditor(components.PostcardEditorDTO{
				Image:     url.GenerateFileUrl("artworks", awid, r.GetString("image"), ""),
				ImageId:   awid,
				Title:     r.GetString("title"),
				Comment:   r.GetString("comment"),
				Technique: r.GetString("technique"),
			}).Render(ctx, c.Response().Writer)

			if err != nil {
				app.Logger().Error(fmt.Sprintf("Failed to render the postcard editor with image_id %s", awid), err)
				return utils.ServerFaultError(c)
			}

			return nil

		})

		e.Router.GET("postcards", func(c echo.Context) error {

			isHtmx := utils.IsHtmxRequest(c)

			postCardId := c.QueryParamDefault("p", "nope")

			if postCardId == "nope" {
				app.Logger().Error(fmt.Sprintf("Invalid postcard id: %s", postCardId))
				return apis.NewBadRequestError("Invalid postcard id", nil)
			}

			cacheKey := fmt.Sprintf("postcard-%s", postCardId)

			if isHtmx {
				cacheKey = cacheKey + ":htmx"
			}

			if app.Store().Has(cacheKey) {
				return c.HTML(http.StatusOK, app.Store().Get(cacheKey).(string))
			}

			var cacheBuffer bytes.Buffer

			r, err := app.Dao().FindRecordById("postcards", postCardId)

			if err != nil {
				app.Logger().Error("Failed to find postcard", "id", postCardId, err)
				return apis.NewNotFoundError("", err)
			}

			if errs := app.Dao().ExpandRecord(r, []string{"image_id"}, nil); len(errs) > 0 {
				app.Logger().Error("Failed to expand record", "id", postCardId, "errors", errs)
				return fmt.Errorf("failed to expand: %v", errs)
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

			ctx := context.Background()

			if isHtmx {
				err = pages.PostcardBlock(content).Render(ctx, &cacheBuffer)
			} else {
				err = pages.PostcardPage(content).Render(ctx, &cacheBuffer)
			}

			if err != nil {
				app.Logger().Error("Failed to render the postcard", err)
				return utils.ServerFaultError(c)
			}

			app.Store().Set(cacheKey, cacheBuffer.String())

			return c.HTML(http.StatusOK, cacheBuffer.String())
		})

		e.Router.POST("postcards", func(c echo.Context) error {

			postData := struct {
				SenderName           string   `json:"sender_name" form:"sender_name" query:"sender_name" validate:"required"`
				SenderEmail          string   `json:"sender_email" form:"sender_email" query:"sender_email" validate:"required,email"`
				Recipients           []string `json:"recipients" form:"recipients[]" query:"recipients" validate:"required"`
				Message              string   `json:"message" form:"message" query:"message" validate:"required"`
				ImageId              string   `json:"image_id" form:"image_id" query:"image_id" validate:"required"`
				NotificationRequired bool     `json:"notification_required" form:"notify_sender" query:"notification_required"`
				RecaptchaToken       string   `json:"recaptcha_token" form:"g-recaptcha-response" query:"recaptcha_token" validate:"required"`
				HoneyPotName         string   `json:"honey_pot_name" form:"name" query:"honey_pot_name"`
				HoneyPotEmail        string   `json:"honey_pot_email" form:"email" query:"honey_pot_email"`
			}{}

			if err := c.Bind(&postData); err != nil {
				utils.SendToastMessage("Failed to parse form", "error", true, c, "")
				return apis.NewBadRequestError("Failed to parse form data", err)
			}

			if postData.HoneyPotEmail != "" || postData.HoneyPotName != "" {
				// this is probably a bot
				app.Logger().Warn("Honey pot triggered", "data", fmt.Sprintf("+%v", postData))
				utils.SendToastMessage("Failed to find postcard collection", "error", true, c, "")
				return nil
			}

			collection, err := app.Dao().FindCollectionByNameOrId("postcards")
			if err != nil {
				app.Logger().Error("Failed to find postcard collection", err)
				utils.SendToastMessage("Failed to find postcard collection", "error", true, c, "")
				return apis.NewNotFoundError("Failed to find postcard collection", err)
			}

			record := models.NewRecord(collection)

			form := forms.NewRecordUpsert(app, record)

			err = form.LoadData(map[string]any{
				"status":        "queued",
				"sender_name":   postData.SenderName,
				"sender_email":  postData.SenderEmail,
				"recipients":    strings.Join(postData.Recipients, ","),
				"message":       p.Sanitize(postData.Message),
				"image_id":      postData.ImageId,
				"notify_sender": postData.NotificationRequired,
			})
			if err != nil {
				app.Logger().Error("Failed to process postcard form", err)
				utils.SendToastMessage("Failed to find postcard collection", "error", true, c, "")
				return apis.NewBadRequestError("Failed to process postcard form", err)
			}

			ctx := context.Background()

			if err := form.Submit(); err != nil {

				r, err := app.Dao().FindRecordById("artworks", postData.ImageId)

				if err != nil {
					app.Logger().Error("Failed to find artwork "+postData.ImageId, err)
					return utils.NotFoundError(c)
				}

				err = components.PostcardEditor(components.PostcardEditorDTO{
					Image:     url.GenerateFileUrl("artworks", postData.ImageId, r.GetString("image"), ""),
					ImageId:   postData.ImageId,
					Title:     r.GetString("title"),
					Comment:   r.GetString("comment"),
					Technique: r.GetString("technique"),
				}).Render(ctx, c.Response().Writer)

				app.Logger().Error(fmt.Sprintf("Failed to store the postcard with image_id %s", postData.ImageId), err)

				utils.SendToastMessage("Failed to store the postcard", "error", false, c, "")

				return nil
			}

			utils.SendToastMessage("Thank you! Your postcard has been queued for sending!", "success", true, c, "")

			return nil
		})

		return nil
	})
}
