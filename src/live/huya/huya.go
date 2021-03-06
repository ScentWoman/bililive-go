package huya

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hr3lxphr6j/bililive-go/src/live"
	"github.com/hr3lxphr6j/bililive-go/src/live/internal"
	"github.com/hr3lxphr6j/bililive-go/src/pkg/http"
	"github.com/hr3lxphr6j/bililive-go/src/pkg/utils"
)

const (
	domain = "www.huya.com"
	cnName = "虎牙"
)

var (
	streamReg = regexp.MustCompile(`"stream": ".*?"`)
)

func init() {
	live.Register(domain, new(builder))
}

type builder struct{}

func (b *builder) Build(url *url.URL) (live.Live, error) {
	return &Live{
		BaseLive: internal.NewBaseLive(url),
	}, nil
}

type Live struct {
	internal.BaseLive
}

func (l *Live) GetInfo() (info *live.Info, err error) {
	dom, err := http.Get(l.Url.String(), nil, nil)
	if err != nil {
		return nil, err
	}
	if res := utils.Match1("哎呀，虎牙君找不到这个主播，要不搜索看看？", string(dom)); res != "" {
		return nil, live.ErrRoomNotExist
	}

	var (
		strFilter = utils.NewStringFilterChain(utils.ParseUnicode, utils.UnescapeHTMLEntity)
		hostName  = strFilter.Do(utils.Match1(`"nick":"([^"]*)"`, string(dom)))
		roomName  = strFilter.Do(utils.Match1(`"introduction":"([^"]*)"`, string(dom)))
		status    = strFilter.Do(utils.Match1(`"isOn":([^,]*),`, string(dom)))
	)

	if hostName == "" || roomName == "" || status == "" {
		return nil, live.ErrInternalError
	}

	info = &live.Info{
		Live:     l,
		HostName: hostName,
		RoomName: roomName,
		Status:   status == "true",
	}
	return info, nil
}

func (l *Live) GetStreamUrls() (us []*url.URL, err error) {
	dom, err := http.Get(l.Url.String(), nil, nil)
	if err != nil {
		return nil, err
	}

	// Decode stream part.
	streamInfo := streamReg.Find(dom)
	if len(streamInfo) < 20 {
		return nil, errors.New("huya.GetStreamUrls: No stream.")
	}
	streamInfo = streamInfo[11 : len(streamInfo)-1]
	streamByte, err := base64.StdEncoding.DecodeString(string(streamInfo))
	if err != nil {
		return nil, err
	}
	streamStr := html.UnescapeString(string(streamByte))

	var (
		sStreamName  = utils.Match1(`"sStreamName":"([^"]*)"`, streamStr)
		sFlvUrl      = strings.ReplaceAll(utils.Match1(`"sFlvUrl":"([^"]*)"`, streamStr), `\/`, `/`)
		sFlvAntiCode = utils.Match1(`"sFlvAntiCode":"([^"]*)"`, streamStr)
		iLineIndex   = utils.Match1(`"iLineIndex":(\d*),`, streamStr)
		uid          = (time.Now().Unix()%1e7*1e6 + int64(1e3*rand.Float64())) % 4294967295
	)
	u, err := url.Parse(fmt.Sprintf("%s/%s.flv", sFlvUrl, sStreamName))
	if err != nil {
		return nil, err
	}
	value := url.Values{}
	value.Add("line", iLineIndex)
	value.Add("p2p", "0")
	value.Add("type", "web")
	value.Add("ver", "1805071653")
	value.Add("uid", fmt.Sprintf("%d", uid))
	u.RawQuery = fmt.Sprintf("%s&%s", value.Encode(), utils.UnescapeHTMLEntity(sFlvAntiCode))
	return []*url.URL{u}, nil
}

func (l *Live) GetPlatformCNName() string {
	return cnName
}
