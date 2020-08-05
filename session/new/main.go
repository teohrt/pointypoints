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
	"github.com/jonsabados/pointypoints/logging"
	"github.com/jonsabados/pointypoints/session"
	"github.com/rs/zerolog"
	"os"
	"time"
)

func NewHandler(prepareLogs logging.Preparer, startSession session.Starter, dispatch api.MessageDispatcher) func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		ctx = prepareLogs(ctx)
		toStart := new(session.StartRequest)
		err := json.Unmarshal([]byte(request.Body), toStart)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Str("error", fmt.Sprintf("%+v", err)).Msg("error session start reading request body")
			return api.NewInternalServerError(ctx), nil
		}
		if toStart.Facilitator.Name == "" {
			return api.NewValidationFailureResponse(ctx, api.ValidationError{
				Errors: []string{"facilitator name is required"},
			}), nil
		}
		toStart.Facilitator.SocketID = request.RequestContext.ConnectionID

		sess, err := startSession(ctx, *toStart)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Str("error", fmt.Sprintf("%+v", err)).Msg("error starting session")
			err = dispatch(ctx, request.RequestContext.ConnectionID, api.Message{
				Type: api.ErrorEncountered,
				Body: err.Error(),
			})
			if err != nil {
				zerolog.Ctx(ctx).Error().Str("error", fmt.Sprintf("%+v", err)).Msg("error dispatching message")
			}
			return api.NewInternalServerError(ctx), nil
		}
		err = dispatch(ctx, request.RequestContext.ConnectionID, api.Message{
			Type: api.SessionCreated,
			Body: sess,
		})
		if err != nil {
			zerolog.Ctx(ctx).Error().Str("error", fmt.Sprintf("%+v", err)).Msg("error dispatching message")
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
	starter := session.NewStarter(dynamo, sessionTable, time.Hour)

	lambda.Start(NewHandler(logPreparer, starter, diutil.NewProdMessageDispatcher()))
}
