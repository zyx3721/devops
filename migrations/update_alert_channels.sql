-- 更新告警配置，添加 邮件 和 钉钉 渠道
-- 请替换为您真实的 Webhook URL 和 Email 地址

UPDATE alert_configs 
SET channels = '[
  {
    "type": "email", 
    "url": "ranpeng@tasiting.com"
  },
  {
    "type": "dingtalk", 
    "url": "https://oapi.dingtalk.com/robot/send?access_token=YOUR_ACCESS_TOKEN",
    "secret": "YOUR_SECRET"
  },
  {
    "type": "webhook",
    "url": "https://open.feishu.cn/open-apis/bot/v2/hook/YOUR_FEISHU_HOOK"
  }
]'
WHERE name = 'CPU_HIGH_ALERT';
