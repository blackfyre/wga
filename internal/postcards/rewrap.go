package postcards

import (
	"errors"
	"fmt"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	defaultTokenRewrapLimit = 100
	maxTokenRewrapLimit     = 1000
)

var (
	errInvalidTokenRewrapSource = errors.New("postcard token rewrap source is invalid")
	errMissingTokenRewrapSource = errors.New("postcard token rewrap source is not configured")
	errActiveTokenRewrapSource  = errors.New("postcard token rewrap source must differ from the active key")
	errTokenRewrapFailed        = errors.New("postcard token rewrap failed")
)

// RewrapTokenKey atomically re-encrypts a bounded, deterministic batch of live
// recipient token envelopes from a configured source key to the active key.
func RewrapTokenKey(app core.App, keyring config.PostcardTokenKeyring, sourceKeyID string, limit int) (int, error) {
	if !validRecipientTokenKeyID(sourceKeyID) {
		return 0, errInvalidTokenRewrapSource
	}
	if _, ok := keyring.Key(sourceKeyID); !ok {
		return 0, errMissingTokenRewrapSource
	}
	activeKeyID, _, err := activeRecipientTokenKey(keyring)
	if err != nil {
		return 0, err
	}
	if sourceKeyID == activeKeyID {
		return 0, errActiveTokenRewrapSource
	}
	if limit <= 0 || limit > maxTokenRewrapLimit {
		return 0, fmt.Errorf("postcard token rewrap limit must be between 1 and %d", maxTokenRewrapLimit)
	}

	rewrapped := 0
	err = app.RunInTransaction(func(txApp core.App) error {
		var rows []struct {
			ID string `db:"id"`
		}
		err := txApp.DB().NewQuery(`
			SELECT id FROM postcard_deliveries
			WHERE status = 'pending'
			AND substr(view_token_envelope, 1, length({:prefix})) = {:prefix}
			ORDER BY id
			LIMIT {:limit}
		`).Bind(dbx.Params{
			"prefix": recipientTokenEnvelopeVersion + "." + sourceKeyID + ".",
			"limit":  limit,
		}).All(&rows)
		if err != nil {
			return errTokenRewrapFailed
		}

		for _, row := range rows {
			delivery, err := txApp.FindRecordById(collectionDeliveries, row.ID)
			if err != nil {
				return errTokenRewrapFailed
			}

			keyID, _, ok := recipientTokenEnvelopeParts(delivery.GetString("view_token_envelope"))
			if !ok || keyID != sourceKeyID {
				return errTokenRewrapFailed
			}

			token, err := recoverRecipientToken(
				keyring,
				delivery.Id,
				delivery.GetString("view_token_envelope"),
				delivery.GetString("view_token_hash"),
			)
			if err != nil {
				return errTokenRewrapFailed
			}
			envelope, err := sealRecipientToken(keyring, delivery.Id, token)
			if err != nil {
				return errTokenRewrapFailed
			}

			result, err := txApp.DB().NewQuery(`
				UPDATE postcard_deliveries
				SET view_token_envelope = {:envelope}
				WHERE id = {:id} AND view_token_envelope = {:previous}
			`).Bind(dbx.Params{
				"envelope": envelope,
				"id":       delivery.Id,
				"previous": delivery.GetString("view_token_envelope"),
			}).Execute()
			if err != nil {
				return errTokenRewrapFailed
			}
			updated, err := result.RowsAffected()
			if err != nil || updated != 1 {
				return errTokenRewrapFailed
			}
			rewrapped++
		}

		if rewrapped == limit {
			return nil
		}
		var controls []struct {
			ID string `db:"id"`
		}
		err = txApp.DB().NewQuery(`
			SELECT id FROM postcard_sender_controls
			WHERE revoked_at = '' AND expires_at > {:now}
			AND substr(token_envelope, 1, length({:prefix})) = {:prefix}
			ORDER BY id LIMIT {:limit}
		`).Bind(dbx.Params{
			"prefix": senderControlTokenEnvelopeVersion + "." + sourceKeyID + ".",
			"limit":  limit - rewrapped,
			"now":    types.NowDateTime(),
		}).All(&controls)
		if err != nil {
			return errTokenRewrapFailed
		}
		for _, row := range controls {
			control, err := txApp.FindRecordById(collectionSenderControls, row.ID)
			if err != nil {
				return errTokenRewrapFailed
			}
			token, err := recoverSenderControlToken(keyring, control.Id, control.GetString("token_envelope"), control.GetString("token_hash"))
			if err != nil {
				return errTokenRewrapFailed
			}
			envelope, err := sealSenderControlToken(keyring, control.Id, token)
			if err != nil {
				return errTokenRewrapFailed
			}
			result, err := txApp.DB().NewQuery(`
				UPDATE postcard_sender_controls SET token_envelope = {:envelope}
				WHERE id = {:id} AND token_envelope = {:previous}
			`).Bind(dbx.Params{"envelope": envelope, "id": control.Id, "previous": control.GetString("token_envelope")}).Execute()
			if err != nil {
				return errTokenRewrapFailed
			}
			updated, err := result.RowsAffected()
			if err != nil || updated != 1 {
				return errTokenRewrapFailed
			}
			rewrapped++
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return rewrapped, nil
}
