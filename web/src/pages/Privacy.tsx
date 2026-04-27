import { Card, Typography } from 'antd'

const { Title, Paragraph, Text } = Typography

export default function Privacy() {
  return (
    <div style={{ minHeight: '100vh', background: '#f5f7fa', padding: '40px 16px' }}>
      <Card style={{ maxWidth: 900, margin: '0 auto' }}>
        <Title level={2}>隐私政策</Title>
        <Paragraph>
          本隐私政策说明我们如何收集、使用、存储及保护您的个人信息。使用本服务即表示您同意本隐私政策的内容。
        </Paragraph>

        <Title level={4}>1. 我们收集的信息</Title>
        <Paragraph>
          我们可能收集账户信息（如邮箱）、设备与日志信息、使用数据、以及企业注册材料文件等。
        </Paragraph>

        <Title level={4}>2. 信息使用目的</Title>
        <Paragraph>
          用于提供与改进服务、身份验证、系统安全、合规审核、客户支持及服务通知。
        </Paragraph>

        <Title level={4}>3. 信息共享与披露</Title>
        <Paragraph>
          我们不会向无关第三方出售您的信息。仅在法律法规要求或经您授权的情况下披露。
        </Paragraph>

        <Title level={4}>4. 数据存储与保留</Title>
        <Paragraph>
          我们在提供服务所需的期限内保存数据，超出期限后将按相关政策进行删除或匿名化处理。
        </Paragraph>

        <Title level={4}>5. 安全措施</Title>
        <Paragraph>
          我们采取合理的技术和管理措施保护数据安全，但无法保证绝对安全。
        </Paragraph>

        <Title level={4}>6. 用户权利</Title>
        <Paragraph>
          您可请求访问、更正或删除个人信息，具体请联系平台客服或企业支持。
        </Paragraph>

        <Title level={4}>7. Cookies 与追踪</Title>
        <Paragraph>
          我们可能使用 Cookies 或类似技术提升体验，您可通过浏览器设置进行管理。
        </Paragraph>

        <Title level={4}>8. 联系方式</Title>
        <Paragraph>
          如对本隐私政策有任何疑问，请联系平台客服或企业支持。
        </Paragraph>

        <Text type="secondary">最后更新日期：2026-02-23</Text>
      </Card>
    </div>
  )
}
