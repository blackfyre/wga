package handlers

import (
	"encoding/json"
	"net/http"

	"blackfyre.ninja/wga/assets"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

func registerPostcardHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("postcard/send", func(c echo.Context) error {

			if !isHtmxRequest(c) {
				return apis.NewBadRequestError("Unexpected request", nil)
			}

			//get awid query param
			awid := c.QueryParam("awid")

			//check if awid is empty
			if awid == "" {
				return apis.NewBadRequestError("awid is empty", nil)
			}

			// find the artwork with the given awid
			// if not found, return 404
			// if found, render the send postcard page with the artwork data
			// if error, return 500

			r, err := app.Dao().FindRecordById("artworks", awid)

			if err != nil {
				return apis.NewNotFoundError("", err)
			}

			html, err := assets.RenderBlock("postcard:editor", map[string]any{
				"Image":     generateFileUrl(app, "artworks", awid, r.GetString("image")),
				"ImageId":   awid,
				"Title":     r.GetString("title"),
				"Comment":   r.GetString("comment"),
				"Technique": r.GetString("technique"),
			})

			if err != nil {
				return apis.NewBadRequestError("", err)
			}

			return c.HTML(http.StatusOK, html)

		})

		e.Router.GET("postcards/:id", func(c echo.Context) error {
			return nil
		})

		e.Router.POST("postcards", func(c echo.Context) error {

			// get the postcard data from the request body
			// validate the postcard data
			// if validation fails, return 400
			// if validation succeeds, create a new postcard record
			// if error, return 500

			postData := struct {
				SenderName           string `json:"sender_name" form:"sender_name" query:"sender_name" validate:"required"`
				SenderEmail          string `json:"sender_email" form:"sender_email" query:"sender_email" validate:"required,email"`
				Recipients           string `json:"recipients" form:"recipients" query:"recipients" validate:"required"`
				Message              string `json:"message" form:"message" query:"message" validate:"required"`
				ImageId              string `json:"image_id" form:"image_id" query:"image_id" validate:"required"`
				NotificationRequired bool   `json:"notification_required" form:"notify_sender" query:"notification_required"`
			}{}

			if err := c.Bind(&postData); err != nil {
				return apis.NewBadRequestError("Failed to read request data", err)
			}

			collection, err := app.Dao().FindCollectionByNameOrId("postcards")
			if err != nil {
				return err
			}

			record := models.NewRecord(collection)

			form := forms.NewRecordUpsert(app, record)

			form.LoadData(map[string]any{
				"status":        "queued",
				"sender_name":   postData.SenderName,
				"sender_email":  postData.SenderEmail,
				"recipients":    postData.Recipients,
				"message":       postData.Message,
				"image_id":      postData.ImageId,
				"notify_sender": postData.NotificationRequired,
			})

			if err := form.Submit(); err != nil {
				return err
			}

			headerData, err := json.Marshal(map[string]any{
				"postcard:dialog:success": map[string]any{
					"message": "Postcard sent successfully!",
				},
			})

			if err != nil {
				return err
			}

			c.Response().Header().Set("HX-Trigger", string(headerData))

			return nil
		})

		return nil
	})
}