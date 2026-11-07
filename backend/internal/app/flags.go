package app

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Question statuses.
const (
	StatusDraft   = "draft"
	StatusActive  = "active"
	StatusFlagged = "flagged"
	StatusRetired = "retired"
)

// FlagQuestion records a Learner's Flag on a Question they answered. One Flag
// pulls the Question for every Learner (status Flagged), which excludes its
// Answers from Mastery; affected Learners' Mastery is recomputed.
func (s *Service) FlagQuestion(ctx context.Context, a Actor, questionID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 2000 {
		return E(KindInvalid, "invalid_reason", "reason must be 1-2000 characters")
	}
	question, err := s.q.GetQuestion(ctx, questionID)
	if err != nil {
		return mapNotFound(err, "question")
	}
	answered, err := s.q.HasAnswered(ctx, store.HasAnsweredParams{LearnerID: a.LearnerID, QuestionID: questionID})
	if err != nil {
		return err
	}
	if !answered {
		return forbidden("you can only Flag a Question you have answered")
	}
	return s.tx(ctx, func(q *store.Queries) error {
		question, err = q.GetQuestionForUpdate(ctx, questionID)
		if err != nil {
			return err
		}
		if _, err := q.CreateFlag(ctx, store.CreateFlagParams{QuestionID: questionID, LearnerID: a.LearnerID, Reason: reason}); err != nil {
			return err
		}
		if question.Status != StatusActive {
			return nil // already pulled
		}
		if err := q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{
			ID: questionID, Status: StatusFlagged, VerificationNote: question.VerificationNote,
		}); err != nil {
			return err
		}
		question.Status = StatusFlagged
		return s.recomputeForQuestion(ctx, q, question, nil)
	})
}

// FlagView is a Flag for the Owner / Content Admin.
type FlagView struct {
	ID             uuid.UUID
	QuestionID     uuid.UUID
	CourseID       uuid.UUID
	LearnerName    string
	Reason         string
	CreatedAt      time.Time
	ResolvedAt     *time.Time
	ResolutionNote *string
	QuestionStatus string
}

func (s *Service) listFlags(ctx context.Context, openOnly bool) ([]FlagView, error) {
	rows, err := s.q.ListFlags(ctx, openOnly)
	if err != nil {
		return nil, err
	}
	out := []FlagView{}
	for i := range rows {
		r := &rows[i]
		out = append(out, FlagView{
			ID: r.ID, QuestionID: r.QuestionID, CourseID: r.CourseID, LearnerName: r.LearnerName, Reason: r.Reason,
			CreatedAt: r.CreatedAt, ResolvedAt: r.ResolvedAt, ResolutionNote: r.ResolutionNote, QuestionStatus: r.QuestionStatus,
		})
	}
	return out, nil
}

// ListFlags lists Flags (Owner only).
func (s *Service) ListFlags(ctx context.Context, a Actor, openOnly bool) ([]FlagView, error) {
	if err := requireOwner(a); err != nil {
		return nil, err
	}
	return s.listFlags(ctx, openOnly)
}

// ContentRequestView is a Content Request.
type ContentRequestView struct {
	ID          uuid.UUID
	ConceptID   uuid.UUID
	ConceptSlug string
	ConceptName string
	CourseID    uuid.UUID
	Reason      string
	CreatedAt   time.Time
	FulfilledAt *time.Time
	Note        *string
}

func (s *Service) listContentRequests(ctx context.Context, openOnly bool) ([]ContentRequestView, error) {
	rows, err := s.q.ListContentRequests(ctx, openOnly)
	if err != nil {
		return nil, err
	}
	out := []ContentRequestView{}
	for i := range rows {
		r := &rows[i]
		out = append(out, ContentRequestView{
			ID: r.ID, ConceptID: r.ConceptID, ConceptSlug: r.ConceptSlug, ConceptName: r.ConceptName, CourseID: r.CourseID,
			Reason: r.Reason, CreatedAt: r.CreatedAt, FulfilledAt: r.FulfilledAt, Note: r.Note,
		})
	}
	return out, nil
}

// ListContentRequests lists open Content Requests (Owner only).
func (s *Service) ListContentRequests(ctx context.Context, a Actor) ([]ContentRequestView, error) {
	if err := requireOwner(a); err != nil {
		return nil, err
	}
	return s.listContentRequests(ctx, true)
}

// ListLearners lists every Learner (Owner only).
func (s *Service) ListLearners(ctx context.Context, a Actor) ([]store.Learner, error) {
	if err := requireOwner(a); err != nil {
		return nil, err
	}
	return s.q.ListLearners(ctx)
}

// RetireQuestionAsOwner retires a Question from the admin UI.
func (s *Service) RetireQuestionAsOwner(ctx context.Context, a Actor, questionID uuid.UUID) error {
	if err := requireOwner(a); err != nil {
		return err
	}
	return s.RetireQuestion(ctx, questionID)
}

// RetireQuestion retires a Question; its Answers stop counting toward Mastery.
func (s *Service) RetireQuestion(ctx context.Context, questionID uuid.UUID) error {
	return s.tx(ctx, func(q *store.Queries) error {
		question, err := q.GetQuestionForUpdate(ctx, questionID)
		if err != nil {
			return mapNotFound(err, "question")
		}
		if question.Status == StatusRetired {
			return nil
		}
		if err := q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{
			ID: questionID, Status: StatusRetired, VerificationNote: question.VerificationNote,
		}); err != nil {
			return err
		}
		if _, err := q.ResolveFlagsForQuestion(ctx, store.ResolveFlagsForQuestionParams{
			QuestionID: questionID, ResolutionNote: ptr("question retired"),
		}); err != nil {
			return err
		}
		question.Status = StatusRetired
		return s.recomputeForQuestion(ctx, q, question, nil)
	})
}

// LearnerPlacementDone reports whether a Learner finished a Placement Test.
func (s *Service) LearnerPlacementDone(ctx context.Context, learnerID uuid.UUID) (bool, error) {
	return s.q.LearnerPlacementDone(ctx, learnerID)
}
