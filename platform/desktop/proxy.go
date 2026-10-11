package desktop

import (
	"bycord/core"
	"context"
	"fmt"
	"time"
)

const DiscordProxyBypass = "cdn.discordapp.com;*.discordapp.net;*.discord.media;*.storage.googleapis.com;<local>"

func SelectProxy(ctx context.Context, proxy, exclude string) (core.Choice, error) {
	excluded := core.ParseExcluded(exclude)
	if len(excluded) == 0 {
		return core.Choice{}, fmt.Errorf("제외 국가 코드를 두 글자로 입력하세요")
	}

	if proxy == "" {
		return core.Resolve(ctx, excluded, nil)
	}

	endpoint, ok := core.ParseProxy(proxy)
	if !ok {
		return core.Choice{}, fmt.Errorf("프록시 주소 형식이 올바르지 않습니다")
	}
	return core.Validate(endpoint, excluded, 6*time.Second)
}
