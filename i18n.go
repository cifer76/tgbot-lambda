package main

import "context"

const (
	UsernameInvalid  = "UsernameInvalid"
	GroupLinkInvalid = "GroupLinkInvalid"
	GroupNotFound    = "GroupNotFound"

	InputGroupLink = "InputGroupLink"
	TopicChoosing  = "TopicChoosing"
	TopicInvalid   = "TopicInvalid"
	TagsInputting  = "TopicInputting"

	IndexFailed  = "IndexFailed"
	IndexSuccess = "IndexSuccess"
)

var (
	texts = map[string]map[string]string{
		UsernameInvalid: {
			"en": "group username in the link invalid, must start with letters and contain only letters, numbers and underscore",
			"zh": "非法的组用户名. 用户名必须以字母开头, 且只包含字母, 数字和下划线",
		},
		GroupLinkInvalid: {
			"en": "Invalid group link, the link must start with https://t.me/ or at least t.me/",
			"zh": "非法的群组链接, 必须以 https://t.me 或 t.me/ 开头",
		},
		GroupNotFound: {
			"en": "find no group or channel, please check your input",
			"zh": "未找到群组或频道, 请检查你的输入",
		},
		InputGroupLink: {
			"en": "please input your group link",
			"zh": "输入群组链接",
		},
		TopicChoosing: {
			"en": "please choose the most relevant topic for your group",
			"zh": "选择一个最符合你的群组的话题",
		},
		TopicInvalid: {
			"en": "group topic invalid, please re-input",
			"zh": "话题输入非法, 请重新输入",
		},
		TagsInputting: {
			"en": "please input your group tags, separated by space",
			"zh": "话题输入非法, 请重新输入",
		},
		IndexFailed: {
			"en": "index failed, please try again later",
			"zh": "收录失败, 请稍后重试",
		},
		IndexSuccess: {
			"en": "%s has been indexed",
			"zh": "已收录 %s",
		},
	}

	startContent = map[string]string{
		"en": `
Input any keyword to search for the related groups.

or choose a command following suit your needs:

/start     - show this information
/index     - index/re-index a group
/list      - list groups by categories
/recommend - recommend some groups
        `,
		"zh": `
你可以直接输入关键词来查找相应的群组.

或者使用如下命令完成不同的操作:

/start     - 展示这个操作指引
/index     - 索引某个群组(或频道)
/list      - 列出所有群组
/recommend - (🔥 推广) 随机推荐一些群组
        `,
	}
)

// TODO embed user settings(language settings) into context and persistently store to db
// thereafter everytime a user initates a chat, get the language settings from db
func getLocalizedText(ctx context.Context, tmpl string) string {
	return texts[tmpl]["zh"]
}

func getStartContent(ctx context.Context) string {
	return startContent["zh"]
}
