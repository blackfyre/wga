package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

// renderPostcardEditor renders the postcard editor HTML for a given artwork ID.
// It takes the artwork ID, a PocketBase app instance, and an Echo context as input.
// It returns the rendered HTML and an error if any occurred.
func renderPostcardEditor(awid string, app *pocketbase.PocketBase, c echo.Context) (string, error) {
	r, err := app.Dao().FindRecordById("artworks", awid)

	if err != nil {
		return "", apis.NewNotFoundError("", err)
	}

	html, err := assets.RenderBlock("postcard:editor", map[string]any{
		"Image":     generateFileUrl(app, "artworks", awid, r.GetString("image")),
		"ImageId":   awid,
		"Title":     r.GetString("title"),
		"Comment":   r.GetString("comment"),
		"Technique": r.GetString("technique"),
	})

	if err != nil {
		return "", apis.NewBadRequestError("", err)
	}

	return html, nil
}

func registerPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("postcard/send", func(c echo.Context) error {

			if !utils.IsHtmxRequest(c) {
				return apis.NewBadRequestError("Unexpected request", nil)
			}

			//get awid query param
			awid := c.QueryParam("awid")

			//check if awid is empty
			if awid == "" {
				return apis.NewBadRequestError("awid is empty", nil)
			}

			html, err := renderPostcardEditor(awid, app, c)

			if err != nil {
				return err
			}

			return c.HTML(http.StatusOK, html)

		})

		e.Router.GET("postcards", func(c echo.Context) error {

			postCardId := c.QueryParamDefault("p", "nope")

			if postCardId == "nope" {
				return apis.NewBadRequestError("Invalid postcard id", nil)
			}

			r, err := app.Dao().FindRecordById("postcards", postCardId)

			if err != nil {
				return apis.NewNotFoundError("", err)
			}

			if errs := app.Dao().ExpandRecord(r, []string{"image_id"}, nil); len(errs) > 0 {
				return fmt.Errorf("failed to expand: %v", errs)
			}

			aw := r.ExpandedOne("image_id")

			data := newTemplateData(c)

			data["SenderName"] = r.GetString("sender_name")
			data["Message"] = r.GetString("message")
			data["AwImage"] = generateFileUrl(app, "artworks", aw.GetString("id"), aw.GetString("image"))
			data["AwTitle"] = aw.GetString("title")
			data["AwComment"] = aw.GetString("comment")
			data["AwTechnique"] = aw.GetString("technique")

			html, err := assets.RenderPage("postcard", data)

			if err != nil {
				return apis.NewBadRequestError("", err)
			}

			return c.HTML(http.StatusOK, html)
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
				sendToastMessage("Failed to parse form", "is-danger", true, c)
				return apis.NewBadRequestError("Failed to parse form data", err)
			}

			if postData.HoneyPotEmail != "" || postData.HoneyPotName != "" {
				// this is probably a bot
				//TODO: use the new generic logger in pb to log this event
				sendToastMessage("Failed to find postcard collection", "is-danger", true, c)
				return nil
			}

			collection, err := app.Dao().FindCollectionByNameOrId("postcards")
			if err != nil {
				sendToastMessage("Failed to find postcard collection", "is-danger", true, c)
				return apis.NewNotFoundError("Failed to find postcard collection", err)
			}

			record := models.NewRecord(collection)

			form := forms.NewRecordUpsert(app, record)

			form.LoadData(map[string]any{
				"status":        "queued",
				"sender_name":   postData.SenderName,
				"sender_email":  postData.SenderEmail,
				"recipients":    strings.Join(postData.Recipients, ","),
				"message":       p.Sanitize(postData.Message),
				"image_id":      postData.ImageId,
				"notify_sender": postData.NotificationRequired,
			})

			if err := form.Submit(); err != nil {

				html, err := renderPostcardEditor(postData.ImageId, app, c)

				if err != nil {
					return err
				}

				sendToastMessage("Failed to store the postcard", "is-danger", false, c)

				return c.HTML(http.StatusOK, html)

			}

			sendToastMessage("Thank you! Your postcard has been queued for sending!", "is-success", true, c)

			return nil
		})

		return nil
	})
}
