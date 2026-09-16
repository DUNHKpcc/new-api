/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package console_setting

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGlobalNotificationsLimitsPublicItems(t *testing.T) {
	notifications := make([]map[string]interface{}, 0, 25)
	for i := 0; i < 25; i++ {
		notifications = append(notifications, map[string]interface{}{
			"id":          i,
			"content":     fmt.Sprintf("Notification %d", i),
			"publishDate": fmt.Sprintf("2026-08-%02dT00:00:00Z", i+1),
		})
	}
	raw, err := common.Marshal(notifications)
	require.NoError(t, err)

	previous := consoleSetting.GlobalNotifications
	consoleSetting.GlobalNotifications = string(raw)
	t.Cleanup(func() { consoleSetting.GlobalNotifications = previous })

	result := GetGlobalNotifications()
	require.Len(t, result, maxPublicNotificationItems)
	require.Equal(t, float64(24), result[0]["id"])
	require.Equal(t, float64(5), result[len(result)-1]["id"])
}

func TestAnnouncementImagePolicySeparatesOrdinaryAndGlobalNotifications(t *testing.T) {
	ordinary := `[{"id":1,"content":"Maintenance","publishDate":"2026-08-01T00:00:00Z","type":"warning","image":"data:image/webp;base64,AAAA"}]`
	normalized, err := NormalizeAnnouncements(ordinary)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"id":1,"content":"Maintenance","publishDate":"2026-08-01T00:00:00Z","type":"warning"}]`, normalized)
	require.NoError(t, ValidateConsoleSettings(ordinary, "Announcements"))

	global := `[{"id":2,"content":"Global notice","publishDate":"2026-08-02T00:00:00Z","type":"success","image":"data:image/webp;base64,AAAA"}]`
	require.NoError(t, ValidateConsoleSettings(global, "GlobalNotifications"))

	previous := GetConsoleSetting().Announcements
	previousGlobal := GetConsoleSetting().GlobalNotifications
	GetConsoleSetting().Announcements = ordinary
	GetConsoleSetting().GlobalNotifications = global
	t.Cleanup(func() {
		GetConsoleSetting().Announcements = previous
		GetConsoleSetting().GlobalNotifications = previousGlobal
	})

	announcements := GetAnnouncements()
	require.Len(t, announcements, 1)
	assert.NotContains(t, announcements[0], "image")
	globalNotifications := GetGlobalNotifications()
	require.Len(t, globalNotifications, 1)
	assert.Equal(t, "data:image/webp;base64,AAAA", globalNotifications[0]["image"])
}
