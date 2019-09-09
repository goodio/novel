// 微信助手
package main

import (
	"github.com/ghaoo/rbootx/adapter/wechat/sdk"
	"github.com/sirupsen/logrus"
	"github.com/ghaoo/novel/wechat"

	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"strings"
)

type assistant struct {
	bot      *wechat.WeChat
}

func NewAssistant(bot *wechat.WeChat) *assistant {
	return &assistant{bot}
}

func (a *assistant) handle(msg wechat.EventMsgData) {
	if msg.IsGroupMsg {

		if msg.MsgType == 10000 && strings.Contains(msg.Content, `加入群聊`) {
			nn, err := search(msg.Content, `"`, `"通过`)
			if err != nil {
				logrus.Errorf(`send group welcome failed %s`, msg.Content)
			}
			a.bot.SendTextMsg(`欢迎【`+nn+`】加入群聊`, msg.FromUserName)
		} else if strings.Contains(msg.Content, `签到`) {
			c := a.bot.ContactByUserName(msg.SenderUserName)
			a.bot.SendTextMsg(fmt.Sprintf(`%s 完成了签到`, c.NickName), msg.FromUserName)
		}

		// 群主踢人
		if msg.IsSendedByMySelf && strings.HasPrefix(msg.Content, `滚蛋`) {

			gun := msg.ToUserName

			nn := strings.Replace(msg.Content, `滚蛋`, ``, 1)
			if members, err := a.bot.MembersOfGroup(gun); err == nil {
				for _, c := range members {
					logrus.Info(c.NickName)
					if c.NickName == nn {
						a.bot.SendTextMsg(nn+` 送你免费飞机票`, gun)
						time.Sleep(3 * time.Second)
						err := a.delMember(gun, c.UserName)
						if err != nil {
							a.bot.SendTextMsg(`暂时不T你吧`, gun)
						}
					}
				}
			}
		}

		if msg.AtMe {

			if strings.Contains(msg.Content, `统计人数`) {
				logrus.Info(msg.ToUserName)
				a.chatRoomMember(msg.ToUserName)
			}
		}
	}
}

func (a *assistant) delMember(groupUserName, memberUserName string) error {
	ps := map[string]interface{}{
		`DelMemberList`: memberUserName,
		`ChatRoomName`:  groupUserName,
		`BaseRequest`:   a.bot.BaseRequest,
	}
	data, _ := json.Marshal(ps)

	url := fmt.Sprintf(`%s/webwxupdatechatroom?fun=delmember`, a.bot.BaseURL)

	var resp wechat.Response

	err := a.bot.Execute(url, bytes.NewReader(data), &resp)

	if err != nil {
		return err
	}

	if resp.IsSuccess() {
		return nil
	}

	return fmt.Errorf(`delete %s on %s failed`, memberUserName, groupUserName)
}


// 统计群里男生和女生数量
func (a *assistant) chatRoomMember(room_name string) (map[string]int, error) {

	stats := make(map[string]int)

	RoomContactList, err := a.bot.MembersOfGroup(room_name)
	if err != nil {
		return nil, err
	}

	man := 0
	woman := 0
	none := 0
	for _, v := range RoomContactList {

		member := a.bot.ContactByUserName(v.UserName)

		if member.Sex == 1 {
			man++
		} else if member.Sex == 2 {
			woman++
		} else {
			none++
		}

	}

	stats = map[string]int{
		"woman": woman,
		"man":   man,
		"none":  none,
	}

	return stats, nil
}

func (a *assistant) addFriend(username, content string) error {
	return a.verifyUser(username, content, 2)
}

// AcceptAddFriend ...
func (a *assistant) acceptAddFriend(username, content string) error {
	return a.verifyUser(username, content, 3)
}

// 添加好友和通过好友验证
func (a *assistant) verifyUser(username, content string, status int) error {

	url := fmt.Sprintf(`%s/webwxverifyuser?r=%s&%s`, a.bot.BaseURL, strconv.FormatInt(time.Now().Unix(), 10), a.bot.PassTicketKV())

	data := map[string]interface{}{
		`BaseRequest`:        a.bot.BaseRequest,
		`Opcode`:             status,
		`VerifyUserListSize`: 1,
		`VerifyUserList`: map[string]string{
			`Value`:            username,
			`VerifyUserTicket`: ``,
		},
		`VerifyContent`:  content,
		`SceneListCount`: 1,
		`SceneList`:      33,
		`skey`:           a.bot.BaseRequest.Skey,
	}

	bs, _ := json.Marshal(data)

	var resp sdk.Response

	err := a.bot.Execute(url, bytes.NewReader(bs), &resp)
	if err != nil {
		return err
	}
	if resp.IsSuccess() {
		return nil
	}
	return resp.Error()
}

// 自动添加好友
func (a *assistant) AutoAcceptAddFirendRequest(msg sdk.MsgData) {
	if msg.MsgType == 37 {
		rInfo := msg.OriginalMsg[`RecommendInfo`].(map[string]interface{})
		err := a.addFriend(rInfo[`UserName`].(string),
			msg.OriginalMsg[`Ticket`].(string))
		if err != nil {
			logrus.Error(err)
		}
		err = a.bot.SendTextMsg(`新添加了一个好友`, `filehelper`)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func search(source, prefix, suffix string) (string, error) {

	index := strings.Index(source, prefix)
	if index == -1 {
		err := fmt.Errorf("can't find [%s] in [%s]", prefix, source)
		return ``, err
	}
	index += len(prefix)

	end := strings.Index(source[index:], suffix)
	if end == -1 {
		err := fmt.Errorf("can't find [%s] in [%s]", suffix, source)
		return ``, err
	}

	result := source[index : index+end]

	return result, nil
}
