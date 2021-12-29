package main

import "context"

const (
	// error messages
	UsernameInvalid  = "UsernameInvalid"
	GroupLinkInvalid = "GroupLinkInvalid"
	GroupNotFound    = "GroupNotFound"
	TopicInvalid     = "TopicInvalid"

	// promptting messages
	InputGroupLink = "InputGroupLink"
	InputTags      = "InputTags"
	TopicChoosing  = "TopicChoosing"

	// result
	IndexFailed  = "IndexFailed"
	IndexSuccess = "IndexSuccess"
)

const (
	InputGroupLinkCN = `
    请输入群组/频道的完整链接或 username.

    e.g. https://t.me/nightyworld
    e.g. nightyworld
    `

	InputTagsCN = `
	为此群组/频道输入几个关键字以使其更容易被发现. 每个群组/频道最多支持 3 个关键字, 以空格分割.

    e.g. 社科 闲聊
    e.g. 消费 数码 geek
    `

	IndexSuccessCN = `
    恭喜! 你的群组/频道已录入.

    群组/频道名: %s
    简介: %s
    检索关键字: %s
    录入时间: %s
    `
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
			"zh": InputGroupLinkCN,
		},
		TopicChoosing: {
			"en": "please choose the most relevant topic for your group",
			"zh": "选择一个最符合你的群组的话题",
		},
		TopicInvalid: {
			"en": "group topic invalid, please re-input",
			"zh": "话题输入非法, 请重新输入",
		},
		InputTags: {
			"en": "please input your group tags, separated by space",
			"zh": InputTagsCN,
		},
		IndexFailed: {
			"en": "index failed, please try again later",
			"zh": "收录失败, 请稍后重试",
		},
		IndexSuccess: {
			"en": "%s has been indexed",
			"zh": IndexSuccessCN,
		},
	}

	startContent = map[string]string{
		"en": `
Input any keyword to search for the related groups.

or choose a command following suit your needs:

/start     - start using / show this help info
/add       - index group
        `,
		"zh": `
收录群组:

TeleEye 机器人提供两种方式收录你的群组

1. 直接将机器人添加为你的群组成员
2. 在机器人对话框使用 /add 命令

搜索群组:

与机器人对话, 直接输入关键词来查找相应的群组

命令列表:

/start     - 开始使用
/add       - 添加群组
        `,
	}
)

var (
	TopicProgramming      = "Programming"
	TopicPolitics         = "Politics"
	TopicEconomics        = "Economics"
	TopicTechnology       = "Technology"
	TopicCryptocurrencies = "Cryptocurrencies"
	TopicBlockchain       = "Blockchain"

	TopicKeyboardTexts = map[string]map[string]string{
		TopicProgramming: {
			"en": "💻 Programming",
			"zh": "💻 编程",
		},
		TopicPolitics: {
			"en": "🏛️ Politics",
			"zh": "🏛️ 政治",
		},
		TopicEconomics: {
			"en": "📈 Economics",
			"zh": "📈 经济金融",
		},
		TopicTechnology: {
			"en": "🖥 Technology",
			"zh": "🖥 科技",
		},
		TopicCryptocurrencies: {
			"en": "₿ Cryptocurrencies",
			"zh": "₿ 加密货币",
		},
		TopicBlockchain: {
			"en": "⛓️ Blockchain",
			"zh": "⛓️ 区块链",
		},
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
