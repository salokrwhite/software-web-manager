import { Card, Typography } from 'antd'

const { Title, Paragraph, Text } = Typography

export default function Terms() {
  return (
    <div style={{ minHeight: '100vh', background: '#f5f7fa', padding: '40px 16px' }}>
      <Card style={{ maxWidth: 900, margin: '0 auto' }}>
        <Title level={2}>服务条款</Title>
        <Paragraph>
          欢迎使用 SWM 软件版本管理平台（以下简称“本服务”）。您在使用本服务前，请务必仔细阅读并理解本服务条款。使用本服务即表示您同意本条款的全部内容。
        </Paragraph>

        <Title level={4}>1. 接受条款</Title>
        <Paragraph>
          您注册、登录或使用本服务，即表示您已阅读、理解并同意受本条款约束。如您不同意，请停止使用本服务。
        </Paragraph>

        <Title level={4}>2. 账户与安全</Title>
        <Paragraph>
          您应保证注册信息真实、准确、完整，并妥善保管账户与密码。因您账户安全问题造成的损失由您自行承担。
        </Paragraph>

        <Title level={4}>3. 服务使用规范</Title>
        <Paragraph>
          您不得利用本服务从事违法违规活动，不得发布或传播违法内容，不得干扰或破坏本服务的正常运行。
        </Paragraph>

        <Title level={4}>4. 费用与支付</Title>
        <Paragraph>
          本服务当前可能提供免费版本或试用服务，具体以平台展示为准。如后续涉及收费，将在相关页面另行说明。
        </Paragraph>

        <Title level={4}>5. 知识产权</Title>
        <Paragraph>
          本服务及其相关内容的知识产权归平台或其权利人所有。未经授权，您不得复制、修改、传播或用于商业用途。
        </Paragraph>

        <Title level={4}>6. 责任限制与免责声明</Title>
        <Paragraph>
          在法律允许的范围内，本平台不对因使用本服务导致的间接、附带或惩罚性损害承担责任。本服务以“现状”提供。
        </Paragraph>

        <Title level={4}>7. 终止与变更</Title>
        <Paragraph>
          平台有权根据业务需要调整或终止服务内容，并将通过公告或通知方式告知。您可随时停止使用本服务。
        </Paragraph>

        <Title level={4}>8. 适用法律与争议解决</Title>
        <Paragraph>
          本条款的解释与争议解决适用相关法律法规。因本服务产生的争议，双方应友好协商解决。
        </Paragraph>

        <Title level={4}>9. 联系方式</Title>
        <Paragraph>
          如对本条款有任何疑问，请联系平台客服或企业支持。
        </Paragraph>

        <Text type="secondary">最后更新日期：2026-02-23</Text>
      </Card>
    </div>
  )
}
