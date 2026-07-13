// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

// ponytail: no update/delete/add-member endpoints — membership and title are
// fixed at creation. Add them if group chats need to grow or be renamed later.

type conversationListBody struct {
	Body Paginated[*models.Conversation]
}

func RegisterConversationRoutes(api huma.API) {
	tags := []string{"discussion"}

	Register(api, huma.Operation{
		OperationID: "conversations-list",
		Summary:     "List conversations",
		Description: "Returns every conversation the authenticated user is a member of, most recently active first.",
		Method:      http.MethodGet, Path: "/conversations", Tags: tags,
	}, conversationsList)

	Register(api, huma.Operation{
		OperationID: "conversations-read",
		Summary:     "Get a conversation",
		Description: "Returns a single conversation. Requires membership.",
		Method:      http.MethodGet, Path: "/conversations/{id}", Tags: tags,
	}, conversationsRead)

	Register(api, huma.Operation{
		OperationID: "conversations-create",
		Summary:     "Start a conversation",
		Description: "Starts a 1:1 or group conversation. member_ids must list at least one other user; the caller is added automatically. is_group is set to true automatically when there's more than one other member.",
		Method:      http.MethodPost, Path: "/conversations", Tags: tags,
	}, conversationsCreate)

	Register(api, huma.Operation{
		OperationID: "conversation-messages-list",
		Summary:     "List messages in a conversation",
		Description: "Returns the messages in one conversation, oldest first, paginated. Requires membership.",
		Method:      http.MethodGet, Path: "/conversations/{conversation}/messages", Tags: tags,
	}, conversationMessagesList)

	Register(api, huma.Operation{
		OperationID: "conversation-messages-create",
		Summary:     "Send a message",
		Description: "Sends a message into a conversation. Requires membership. Bumps the conversation to the top of the list.",
		Method:      http.MethodPost, Path: "/conversations/{conversation}/messages", Tags: tags,
	}, conversationMessagesCreate)
}

func init() { AddRouteRegistrar(RegisterConversationRoutes) }

func conversationsList(ctx context.Context, in *struct {
	ListParams
}) (*conversationListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.Conversation{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.Conversation)
	if !ok {
		return nil, fmt.Errorf("conversations.ReadAll returned unexpected type %T", result)
	}
	return &conversationListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func conversationsRead(ctx context.Context, in *struct {
	ID int64 `path:"id"`
	conditional.Params
}) (*singleReadBody[models.Conversation], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	conversation := &models.Conversation{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, conversation, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return conditionalReadResponse(&in.Params, conversation, conversation.Updated, maxPermission)
}

func conversationsCreate(ctx context.Context, in *struct {
	Body models.Conversation
}) (*singleBody[models.Conversation], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Conversation]{Body: &in.Body}, nil
}

type chatMessageListBody struct {
	Body Paginated[*models.ChatMessage]
}

func conversationMessagesList(ctx context.Context, in *struct {
	ConversationID int64 `path:"conversation" doc:"The id of the conversation whose messages to list."`
	ListParams
}) (*chatMessageListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.ChatMessage{ConversationID: in.ConversationID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.ChatMessage)
	if !ok {
		return nil, fmt.Errorf("chatMessages.ReadAll returned unexpected type %T", result)
	}
	return &chatMessageListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func conversationMessagesCreate(ctx context.Context, in *struct {
	ConversationID int64 `path:"conversation" doc:"The id of the conversation to send the message into."`
	Body           models.ChatMessage
}) (*singleBody[models.ChatMessage], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	in.Body.ConversationID = in.ConversationID
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.ChatMessage]{Body: &in.Body}, nil
}
