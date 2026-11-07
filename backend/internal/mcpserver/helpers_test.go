package mcpserver_test

import (
	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

func appQuiz(e *testutil.Env) app.CreateSessionInput {
	c := e.Concepts["a"]
	return app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: &c}
}

func appAnswer(session, question, choice uuid.UUID) app.SubmitAnswerInput {
	return app.SubmitAnswerInput{SessionID: session, QuestionID: question, ChoiceID: choice}
}
