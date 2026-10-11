package dao

import (
	"context"
	"database/sql"

	"github.com/nexsoft-git/nexlogger/log"
)

// FavoriteDAO is the only thing that reads or writes favorite_items.
//
// It returns raw errors for anything unexpected (the caller renders those as a
// generic 500) and uses return values — AddOutcome, the removed bool — for the
// business conditions the caller must distinguish.
type FavoriteDAO interface {
	// Add records a favorite for (userID, itemID), enforcing the per-user cap
	// atomically. limit is supplied by the caller so the policy stays in the
	// service and only the mechanism lives here.
	Add(userID, itemID, limit int64) (AddOutcome, error)

	// Remove deletes exactly the (userID, itemID) row, in one scoped statement.
	// removed is false when no such row exists — which is the same answer for
	// "no such favorite" and "that favorite belongs to someone else", and
	// deliberately so: the caller cannot tell them apart (REQ-008).
	Remove(userID, itemID int64) (removed bool, err error)

	// List returns the user's favorites with each item's current catalog state.
	List(userID int64) ([]FavoriteItem, error)

	// ItemExists reports whether the catalog item exists at all. Used to answer
	// a bad item_id with a real 404 instead of letting the foreign key raise a
	// constraint violation, which the formator would turn into a generic 500.
	ItemExists(itemID int64) (exists bool, err error)
}

type favoriteDAO struct {
	db *sql.DB
}

func NewFavoriteDAO(db *sql.DB) FavoriteDAO {
	return &favoriteDAO{db: db}
}

func (d *favoriteDAO) Add(userID, itemID, limit int64) (AddOutcome, error) {
	ctx := context.Background()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when begin transaction for add favorite")
		return AddInserted, err
	}
	// Rollback is a no-op once Commit has succeeded; this covers every early
	// return and the panic path.
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Error().Err(err).Caller().Msg("error when rollback add favorite transaction")
		}
	}()

	// Lock this user's row for the rest of the transaction. Without it the
	// count-then-insert below is a bare pre-check, and two concurrent requests
	// can both read 499 and both insert, landing the user at 501 — REQ-007 says
	// the 501st must fail loudly, not silently succeed.
	// ponytail: serialize per user, not globally; the lock is held for three
	// fast indexed statements, so contention is nil at this scale.
	var lockedUserID int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID,
	).Scan(&lockedUserID)
	if err == sql.ErrNoRows {
		// The token validator supplies the user id from the token, not from this
		// table, so a valid token for a user row that no longer exists lands
		// here. That is a genuine inconsistency, not a caller error — pass it
		// through and let the formator answer with a generic 500.
		log.Error().Int64("user_id", userID).Caller().Msg("add favorite for unknown user")
		return AddInserted, err
	}
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when lock user for add favorite")
		return AddInserted, err
	}

	// Already favorited? Checked before the cap, because re-favoriting an item
	// the user already has is idempotent success (REQ-005) even when they are
	// sitting at the cap — the cap is on *distinct* favorites.
	var alreadyFavorited bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM favorite_items WHERE user_id = $1 AND item_id = $2)`,
		userID, itemID,
	).Scan(&alreadyFavorited)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when check existing favorite")
		return AddInserted, err
	}
	if alreadyFavorited {
		return AddAlreadyFavorited, nil
	}

	var count int64
	err = tx.QueryRowContext(ctx,
		`SELECT count(*) FROM favorite_items WHERE user_id = $1`, userID,
	).Scan(&count)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when count favorites for cap check")
		return AddInserted, err
	}
	if count >= limit {
		return AddLimitReached, nil
	}

	// ON CONFLICT DO NOTHING keeps the composite primary key as the actual
	// guarantee, not the checks above — they only produce the friendly outcome.
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO favorite_items (user_id, item_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, item_id) DO NOTHING`,
		userID, itemID,
	); err != nil {
		log.Error().Err(err).Caller().Msg("error when insert favorite")
		return AddInserted, err
	}

	if err = tx.Commit(); err != nil {
		log.Error().Err(err).Caller().Msg("error when commit add favorite transaction")
		return AddInserted, err
	}

	return AddInserted, nil
}

func (d *favoriteDAO) Remove(userID, itemID int64) (bool, error) {
	// Both keys in the one statement — never "fetch, then check the owner in Go".
	// Zero rows affected is then the correct and only answer for both a missing
	// row and someone else's row (REQ-008).
	result, err := d.db.ExecContext(context.Background(),
		`DELETE FROM favorite_items WHERE user_id = $1 AND item_id = $2`,
		userID, itemID,
	)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when delete favorite")
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when read rows affected for delete favorite")
		return false, err
	}

	return affected > 0, nil
}

func (d *favoriteDAO) List(userID int64) ([]FavoriteItem, error) {
	// LEFT JOIN, not INNER: REQ-004 says a favorited item must still list when it
	// is discontinued. The spec's premise is that catalog_items rows are never
	// hard-deleted, so both joins behave identically today — but if that premise
	// is ever broken, INNER JOIN drops the row silently, which is precisely the
	// failure REQ-004 forbids. LEFT JOIN degrades to "unavailable" instead.
	rows, err := d.db.QueryContext(context.Background(),
		`SELECT f.item_id,
		        COALESCE(c.name, ''),
		        COALESCE(c.is_available, FALSE),
		        f.created_at
		   FROM favorite_items f
		   LEFT JOIN catalog_items c ON c.id = f.item_id
		  WHERE f.user_id = $1
		  ORDER BY f.created_at DESC, f.item_id ASC`,
		userID,
	)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when query favorites list")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Err(err).Caller().Msg("error when close favorites rows")
		}
	}()

	favorites := make([]FavoriteItem, 0)
	for rows.Next() {
		var item FavoriteItem
		if err := rows.Scan(&item.ItemID, &item.Name, &item.Available, &item.FavoritedAt); err != nil {
			log.Error().Err(err).Caller().Msg("error when scan favorite row")
			return nil, err
		}
		favorites = append(favorites, item)
	}
	if err := rows.Err(); err != nil {
		log.Error().Err(err).Caller().Msg("error when iterate favorites rows")
		return nil, err
	}

	return favorites, nil
}

func (d *favoriteDAO) ItemExists(itemID int64) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM catalog_items WHERE id = $1)`, itemID,
	).Scan(&exists)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when check catalog item existence")
		return false, err
	}

	return exists, nil
}
