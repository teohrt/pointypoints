package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/jonsabados/pointypoints/api"
	"github.com/jonsabados/pointypoints/diutil"
	"github.com/jonsabados/pointypoints/lock"
	"github.com/jonsabados/pointypoints/logging"
	"github.com/jonsabados/pointypoints/session"
	"github.com/rs/zerolog"
	"os"
	"time"
)

func NewHandler(prepareLogs logging.Preparer, loadSession session.Loader, locker lock.GlobalLockAppropriator, saveSession session.Saver) func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		ctx = prepareLogs(ctx)
		r := new(session.ShowVotesRequest)
		err := json.Unmarshal([]byte(request.Body), r)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Str("error", fmt.Sprintf("%+v", err)).Msg("error reading load request body")
			return api.NewInternalServerError(ctx), nil
		}

		recordLock, err := locker(ctx, lock.SessionLockKey(r.SessionID))
		if err != nil {
			zerolog.Ctx(ctx).Error().Str("error", fmt.Sprintf("%+v", err)).Msg("error locking session")
			return api.NewInternalServerError(ctx), nil
		}
		defer func() {
			err := recordLock.Unlock(ctx)
			if err != nil {
				zerolog.Ctx(ctx).Error().Str("error", fmt.Sprintf("%+v", err)).Msg("unable to release lock")
			}
		}()
		sess, err := loadSession(ctx, r.SessionID)
		if err != nil {
			zerolog.Ctx(ctx).Error().Str("error", fmt.Sprintf("%+v", err)).Msg("error reading session")
			return api.NewPermissionDeniedResponse(ctx), nil
		}
		if sess == nil {
			zerolog.Ctx(ctx).Warn().Str("sessionID", r.SessionID).Msg("session not found")
			return api.NewPermissionDeniedResponse(ctx), nil
		}
		if sess.FacilitatorSessionKey != r.FacilitatorSessionKey {
			zerolog.Ctx(ctx).Warn().Str("sessionID", r.SessionID).Msg("attempt to show votes with incorrect facilitator key")
			return api.NewPermissionDeniedResponse(ctx), nil
		}

		sess.VotesShown = false
		for i := 0; i < len(sess.Participants); i++ {
			sess.Participants[i].CurrentVote = nil
		}

		err = saveSession(ctx, *sess)
		if err != nil {
			zerolog.Ctx(ctx).Error().Str("error", fmt.Sprintf("%+v", err)).Msg("error saving session")
			return api.NewInternalServerError(ctx), nil
		}
		return api.NewSuccessResponse(ctx, sess), nil
	}
}

func main() {
	err := xray.Configure(xray.Config{
		LogLevel: "warn",
	})
	if err != nil {
		panic(err)
	}

	logPreparer := logging.NewPreparer()
	sess, err := awssession.NewSession(&aws.Config{})
	if err != nil {
		panic(err)
	}
	dynamo := dynamodb.New(sess)
	xray.AWS(dynamo.Client)

	sessionTable := os.Getenv("SESSION_TABLE")
	loader := session.NewLoader(dynamo, sessionTable)

	lockTable := os.Getenv("LOCK_TABLE")
	locker := lock.NewGlobalLockAppropriator(dynamo, lockTable, time.Millisecond*5, time.Second)

	watcherTable := os.Getenv("WATCHER_TABLE")
	notifier := session.NewChangeNotifier(dynamo, watcherTable, diutil.NewProdMessageDispatcher())

	saveSess := session.NewSaver(dynamo, sessionTable, notifier, time.Hour)

	lambda.Start(NewHandler(logPreparer, loader, locker, saveSess))
}
