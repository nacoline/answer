/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package notification

import (
	"context"
	"time"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/schema"
	"github.com/segmentfault/pacman/i18n"
	"github.com/segmentfault/pacman/log"
)

func (ns *ExternalNotificationService) handleNewCommentNotification(ctx context.Context,
	msg *schema.ExternalNotificationMsg) error {
	log.Debugf("try to send new comment notification %+v", msg)

	notificationConfig, exist, err := ns.userNotificationConfigRepo.GetByUserIDAndSource(ctx, msg.ReceiverUserID, constant.InboxSource)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}
	channels := schema.NewNotificationChannelsFormJson(notificationConfig.Channels)
	for _, channel := range channels {
		if !channel.Enable {
			continue
		}
		switch channel.Key {
		case constant.EmailChannel:
			ns.sendNewCommentNotificationEmail(ctx, msg.ReceiverUserID, msg.ReceiverEmail, msg.ReceiverLang, msg.NewCommentTemplateRawData)
		}
	}
	return nil
}

func (ns *ExternalNotificationService) sendNewCommentNotificationEmail(ctx context.Context,
	userID, email, lang string, rawData *schema.NewCommentTemplateRawData) {
	if unavailable := ns.checkUserStatusBeforeNotification(ctx, userID); unavailable {
		return
	}
	codeContent := &schema.EmailCodeContent{
		SourceType: schema.UnsubscribeSourceType,
		NotificationSources: []constant.NotificationSource{
			constant.InboxSource,
		},
		Email:                    email,
		UserID:                   userID,
		SkipValidationLatestCode: true,
	}
	// If receiver has set language, use it to send email.
	if len(lang) > 0 {
		ctx = context.WithValue(ctx, constant.AcceptLanguageFlag, i18n.Language(lang))
	}
	title, body, err := ns.emailService.NewCommentTemplate(ctx, rawData)
	if err != nil {
		log.Error(err)
		return
	}

	ns.emailService.SendAndSaveCodeWithTime(
		ctx, userID, email, title, body, rawData.UnsubscribeCode, codeContent.ToJSONString(), 1*24*time.Hour)
}
