package favorite

import (
	stderrors "errors"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"

	"github.com/william22913/sandbox-svc-code/dao"
	"github.com/william22913/sandbox-svc-code/dto/in"
	svcout "github.com/william22913/sandbox-svc-code/dto/out"
	svcerror "github.com/william22913/sandbox-svc-code/error"
)

// Add handles POST /favorites.
//
// Returns the ordinary error (a generic 500) for anything unexpected, and a
// typed sentinel for the two outcomes the caller is meant to understand: the
// item does not exist (404) and the user is at the cap (422).
func (s *Service) Add(
	ctx *internalCtx.ContextModel,
	_ model.URLParam,
	dto interface{},
) (map[string]string, interface{}, error) {

	// The body DTO is this service's own GetDTO(), so this always holds. The
	// comma-ok form is still used deliberately: a bare assertion on a wrong type
	// panics, and a panic inside the controller's wrapper is recovered, logged,
	// and then never answered — the client waits for the read timeout instead of
	// getting an error.
	req, ok := dto.(*in.AddFavoriteRequest)
	if !ok {
		log.Error().Caller().Msgf("add favorite received unexpected dto type %T", dto)
		// A plain error, not a sentinel: nothing is actionable for the caller, so
		// the formator's generic 500 is the honest answer.
		return nil, nil, stderrors.New("unexpected dto type")
	}

	// The user comes from the access token, never from the request — that is
	// REQ-008 satisfied by construction rather than by a check.
	userID := ctx.AuthAccessTokenModel.AuthenticationServerUserID

	// architecture.md assigns the existence check to this method. Without it a
	// bad item_id reaches the insert and the foreign key raises a constraint
	// violation, which the formator can only report as a generic 500.
	exists, err := s.dao.ItemExists(req.ItemID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, svcerror.ErrCatalogItemNotFound
	}

	outcome, err := s.dao.Add(userID, req.ItemID, MaxFavoritesPerUser)
	if err != nil {
		return nil, nil, err
	}

	switch outcome {
	case dao.AddInserted:
		return nil, svcout.AddFavoriteResponse{ItemID: req.ItemID}, nil

	case dao.AddAlreadyFavorited:
		// REQ-005: a repeat favorite is a success, not an error and not a
		// duplicate row. AlreadyFavorited tells the client which of the two it
		// got, without changing the status code.
		return nil, svcout.AddFavoriteResponse{ItemID: req.ItemID, AlreadyFavorited: true}, nil

	case dao.AddLimitReached:
		// REQ-007: a clear error, with nothing inserted and nothing evicted.
		return nil, nil, svcerror.ErrFavoritesLimitReached
	}

	// An outcome this switch does not know about means the DAO and the service
	// have drifted apart. Answering 500 is honest; inventing a success is not.
	log.Error().Caller().Msgf("add favorite returned unknown outcome %v", outcome)
	return nil, nil, stderrors.New("unknown add outcome")
}
