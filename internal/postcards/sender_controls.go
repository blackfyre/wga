package postcards

import (
	"errors"
	"strings"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

var ErrSenderControlUnavailable = errors.New("postcard sender control is unavailable")

type SenderControlAccess struct {
	ControlID  string
	PostcardID string
	Token      string
	ExpiresAt  types.DateTime
}

type SenderControl struct {
	ID       string
	Postcard *core.Record
}

type SenderDeliveryStatus struct {
	Recipient string
	State     string
}

type SenderStatus struct {
	PostcardID string
	State      string
	Deliveries []SenderDeliveryStatus
}

func SenderStatusForControl(app core.App, control *SenderControl) (*SenderStatus, error) {
	if control == nil || control.Postcard == nil {
		return nil, ErrSenderControlUnavailable
	}
	deliveries, err := app.FindRecordsByFilter(collectionDeliveries, "postcard = {:postcard}", "+id", 0, 0, map[string]any{"postcard": control.Postcard.Id})
	if err != nil {
		return nil, ErrSenderControlUnavailable
	}
	status := &SenderStatus{PostcardID: control.Postcard.Id, State: control.Postcard.GetString("status"), Deliveries: make([]SenderDeliveryStatus, 0, len(deliveries))}
	for _, delivery := range deliveries {
		status.Deliveries = append(status.Deliveries, SenderDeliveryStatus{Recipient: maskSenderRecipient(delivery.GetString("recipient")), State: delivery.GetString("status")})
	}
	return status, nil
}

func maskSenderRecipient(value string) string {
	at := strings.LastIndex(value, "@")
	if at <= 0 || at == len(value)-1 || strings.HasPrefix(value, "purged:") {
		return "recipient"
	}
	return string([]rune(value[:at])[0]) + "••••@" + value[at+1:]
}

// FindSenderControl authorises a sender request by its presented bearer token.
func FindSenderControl(app core.App, token string, now types.DateTime) (*SenderControl, error) {
	if !ValidRecipientToken(token) {
		return nil, ErrSenderControlUnavailable
	}
	control, err := app.FindFirstRecordByFilter(collectionSenderControls, "token_hash = {:hash} && expires_at > {:now} && revoked_at = ''", map[string]any{"hash": HashRecipientToken(token), "now": now})
	if err != nil {
		return nil, ErrSenderControlUnavailable
	}
	postcard, err := app.FindRecordById(collectionPostcards, control.GetString("postcard"))
	if err != nil || postcard.GetDateTime("retention_until").Before(now) {
		return nil, ErrSenderControlUnavailable
	}
	return &SenderControl{ID: control.Id, Postcard: postcard}, nil
}

func IssueSenderControl(app core.App, keyring config.PostcardTokenKeyring, postcardID string, now types.DateTime) (*SenderControlAccess, error) {
	if _, _, err := activeRecipientTokenKey(keyring); err != nil {
		return nil, err
	}
	access := &SenderControlAccess{PostcardID: postcardID}
	err := app.RunInTransaction(func(txApp core.App) error {
		postcard, err := txApp.FindRecordById(collectionPostcards, postcardID)
		if err != nil {
			return ErrSenderControlUnavailable
		}
		retentionUntil := postcard.GetDateTime("retention_until")
		if retentionUntil.IsZero() || !retentionUntil.After(now) {
			return ErrSenderControlUnavailable
		}
		access.ExpiresAt = now.Add(RecipientTokenValidity)
		if retentionUntil.Before(access.ExpiresAt) {
			access.ExpiresAt = retentionUntil
		}
		controls, err := txApp.FindCollectionByNameOrId(collectionSenderControls)
		if err != nil {
			return err
		}
		token, err := newRecipientToken()
		if err != nil {
			return err
		}
		control := core.NewRecord(controls)
		control.Set("postcard", postcardID)
		control.Set("token_hash", HashRecipientToken(token))
		control.Set("expires_at", access.ExpiresAt)
		if err := txApp.Save(control); err != nil {
			return err
		}
		envelope, err := sealSenderControlToken(keyring, control.Id, token)
		if err != nil {
			return err
		}
		control.Set("token_envelope", envelope)
		if err := txApp.Save(control); err != nil {
			return err
		}
		access.ControlID = control.Id
		access.Token = token
		return nil
	})
	if err != nil {
		return nil, err
	}
	return access, nil
}

func RecoverSenderControl(app core.App, keyring config.PostcardTokenKeyring, postcardID string, now types.DateTime) (*SenderControlAccess, error) {
	control, err := app.FindFirstRecordByFilter(collectionSenderControls, "postcard = {:postcard} && expires_at > {:now} && revoked_at = ''", map[string]any{"postcard": postcardID, "now": now})
	if err != nil {
		return nil, ErrSenderControlUnavailable
	}
	token, err := recoverSenderControlToken(keyring, control.Id, control.GetString("token_envelope"), control.GetString("token_hash"))
	if err != nil {
		return nil, ErrSenderControlUnavailable
	}
	return &SenderControlAccess{ControlID: control.Id, PostcardID: postcardID, Token: token, ExpiresAt: control.GetDateTime("expires_at")}, nil
}
