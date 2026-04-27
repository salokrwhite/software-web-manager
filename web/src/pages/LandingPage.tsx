import { useState, useEffect } from 'react'
import { Alert, Button, Card, Row, Col, Typography, Space, Badge, Drawer, Grid, theme } from 'antd'
import {
  RocketOutlined,
  SafetyOutlined,
  BarChartOutlined,
  CloudOutlined,
  CheckCircleOutlined,
  ArrowRightOutlined,
  GithubOutlined,
  TeamOutlined,
  GlobalOutlined,
  ApiOutlined,
  ArrowUpOutlined,
  MenuOutlined
} from '@ant-design/icons'
import i18next from 'i18next'
import { initReactI18next, useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'

const { Title, Paragraph, Text } = Typography
const { useToken } = theme
type LandingLanguage = 'zh' | 'en'

const LANDING_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialLandingLanguage = (): LandingLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(LANDING_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const landingResources = {
  zh: {
    translation: {
      nav: {
        menuAriaLabel: '打开导航菜单',
        drawerTitle: '导航菜单',
        home: '首页',
        product: '功能介绍',
        pricing: '定价方案',
        changelog: '更新日志',
        apiDocs: 'API 文档',
        login: '登录',
        freeTrial: '免费试用'
      },
      hero: {
        titleLine1: '企业级软件版本',
        titleHighlight: '全生命周期管理',
        descriptionLine1: '由小白科学院自研的统一软件版本管理平台，支持版本发布、灰度/预览、回滚、强更/可选更新、',
        descriptionLine2: '渠道分发与数据分析，让版本管理更简单、更安全、更智能',
        startNow: '立即开始',
        viewDocs: '查看文档'
      },
      stats: [
        { value: '99.9%', label: '服务可用性' },
        { value: '50ms', label: '平均响应' },
        { value: '10M+', label: '日更新次数' },
        { value: '1000+', label: '企业用户' }
      ],
      coreFeatures: {
        title: '核心功能',
        subtitle: '一站式解决软件版本管理的所有痛点',
        items: [
          {
            title: '版本发布管理',
            description: '支持多平台版本发布，一键部署到灰度环境，快速验证后全量发布'
          },
          {
            title: '灰度与回滚',
            description: '智能灰度发布策略，支持按用户、地域、设备维度灰度，秒级回滚保障'
          },
          {
            title: '数据分析',
            description: '实时监控版本覆盖率、更新成功率、用户反馈，数据驱动决策'
          },
          {
            title: '多渠道分发',
            description: '支持应用商店、企业分发、OTA 等多种渠道，灵活配置分发策略'
          },
          {
            title: '多语言 SDK',
            description: '提供 Go、Python、C++ 等多语言 SDK，快速集成到现有项目'
          },
          {
            title: '团队协作',
            description: '多组织、多应用管理，细粒度权限控制，支持 RBAC 权限模型'
          }
        ]
      },
      highlightsSection: {
        title: '为什么选择 SWM？',
        description: '我们深入了解企业在软件版本管理中的痛点，提供专业的解决方案，帮助团队提升发布效率，降低版本风险。',
        items: [
          '支持强更/可选更新策略配置',
          '版本 diff 增量更新',
          '自定义更新弹窗 UI',
          'A/B 测试版本对比',
          '崩溃率实时监控',
          '自动化 CI/CD 集成'
        ],
        learnMore: '了解更多'
      },
      globalSection: {
        title: '全球化部署支持',
        description: '支持多地域部署，就近访问，全球 CDN 加速，确保用户在任何地方都能获得极速的更新体验。',
        nodes: '全球节点',
        continents: '大洲覆盖',
        latency: '平均延迟'
      },
      pricing: {
        title: '套餐方案',
        subtitle: '根据团队规模选择合适的方案，灵活升级',
        free: {
          feature1: '适用于个人开发者，快速上手验证需求',
          feature2: '创建应用上限 10 个',
          feature3: '基础功能齐全，适合试用与原型验证',
          cta: '开始使用'
        },
        team: {
          feature1: '适合中小团队开发，支持团队协作',
          feature2: '支持部门管理与跨部门协作',
          feature3: '适配研发流程，提升发布与协作效率',
          cta: '开始使用'
        },
        enterprise: {
          feature1: '适合大型企业，支持私有部署',
          feature2: '包含前两档能力并支持审计',
          feature3: '更高安全与合规保障，满足复杂治理需求',
          cta: '联系我们'
        }
      },
      cta: {
        title: '准备好提升您的版本管理效率了吗？',
        description: '立即开始使用 SWM，享受免费的基础版本，无需信用卡',
        start: '免费开始使用',
        switchToEnglish: 'English',
        switchToChinese: '中文',
        contactSales: '联系销售'
      },
      footer: {
        description: '企业级软件版本管理平台，让版本管理更简单、更安全、更智能。',
        product: '产品',
        support: '支持',
        contactUs: '联系我们',
        serviceStatus: '服务状态',
        copyright: '© 1996-2024 小白科学研究院 All rights reserved. 吉ICP备2025026240号-3'
      }
    }
  },
  en: {
    translation: {
      nav: {
        menuAriaLabel: 'Open navigation menu',
        drawerTitle: 'Navigation',
        home: 'Home',
        product: 'Features',
        pricing: 'Pricing',
        changelog: 'Changelog',
        apiDocs: 'API Docs',
        login: 'Login',
        freeTrial: 'Free Trial'
      },
      hero: {
        titleLine1: 'Enterprise Software',
        titleHighlight: 'Lifecycle Management',
        descriptionLine1: 'An in-house software version platform from Xiaobai Institute, supporting release, canary/preview, rollback, and mandatory/optional updates,',
        descriptionLine2: 'plus channel distribution and analytics to make version management simpler, safer, and smarter.',
        startNow: 'Get Started',
        viewDocs: 'View Docs'
      },
      stats: [
        { value: '99.9%', label: 'Service Availability' },
        { value: '50ms', label: 'Avg Response' },
        { value: '10M+', label: 'Daily Updates' },
        { value: '1000+', label: 'Enterprise Customers' }
      ],
      coreFeatures: {
        title: 'Core Features',
        subtitle: 'One platform to solve key software version management challenges',
        items: [
          {
            title: 'Release Management',
            description: 'Publish to multiple platforms, deploy to canary in one click, validate fast, then roll out globally.'
          },
          {
            title: 'Canary and Rollback',
            description: 'Smart canary strategy by user, region, and device with second-level rollback protection.'
          },
          {
            title: 'Analytics',
            description: 'Monitor adoption, update success rate, and feedback in real time to drive decisions.'
          },
          {
            title: 'Multi-channel Distribution',
            description: 'Support app stores, enterprise channels, OTA, and flexible distribution policies.'
          },
          {
            title: 'Multi-language SDK',
            description: 'Built-in Go, Python, C++ and more for quick integration with existing projects.'
          },
          {
            title: 'Team Collaboration',
            description: 'Multi-organization and multi-app management with fine-grained RBAC permissions.'
          }
        ]
      },
      highlightsSection: {
        title: 'Why SWM?',
        description: 'We deeply understand version management pain points and provide practical solutions that improve release efficiency and reduce risk.',
        items: [
          'Support mandatory and optional update policies',
          'Version diff incremental updates',
          'Custom update dialog UI',
          'A/B testing between versions',
          'Real-time crash-rate monitoring',
          'Automated CI/CD integration'
        ],
        learnMore: 'Learn More'
      },
      globalSection: {
        title: 'Global Deployment Support',
        description: 'Deploy across regions, access nearby, and accelerate with global CDN for consistently fast updates worldwide.',
        nodes: 'Global Nodes',
        continents: 'Continents',
        latency: 'Avg Latency'
      },
      pricing: {
        title: 'Plans',
        subtitle: 'Choose the right plan for your team and scale as you grow',
        free: {
          feature1: 'For individual developers to get started and validate ideas quickly',
          feature2: 'Up to 10 applications',
          feature3: 'Complete baseline capabilities for trial and prototyping',
          cta: 'Start'
        },
        team: {
          feature1: 'For small to medium teams with collaboration support',
          feature2: 'Department management and cross-team collaboration',
          feature3: 'Fits engineering workflows to improve release efficiency',
          cta: 'Start'
        },
        enterprise: {
          feature1: 'For large enterprises with private deployment support',
          feature2: 'Includes previous tiers with auditing support',
          feature3: 'Higher security and compliance for complex governance',
          cta: 'Contact Us'
        }
      },
      cta: {
        title: 'Ready to improve release efficiency?',
        description: 'Start using SWM now with the free tier, no credit card required.',
        start: 'Start for Free',
        switchToEnglish: 'English',
        switchToChinese: '中文',
        contactSales: 'Contact Sales'
      },
      footer: {
        description: 'Enterprise software version management platform: simpler, safer, smarter.',
        product: 'Product',
        support: 'Support',
        contactUs: 'Contact Us',
        serviceStatus: 'Service Status',
        copyright: '© 1996-2024 Xiaobai Institute. All rights reserved. ICP 吉ICP备2025026240号-3'
      }
    }
  }
} as const

if (!i18next.isInitialized) {
  void i18next.use(initReactI18next).init({
    resources: landingResources,
    lng: getInitialLandingLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', landingResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', landingResources.en.translation, true, true)
}

export default function LandingPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { token } = useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [scrolled, setScrolled] = useState(false)
  const [showBackTop, setShowBackTop] = useState(false)
  const [pageAnnouncementEnabled, setPageAnnouncementEnabled] = useState(false)
  const [pageAnnouncementContent, setPageAnnouncementContent] = useState('')
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const currentLanguage: LandingLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const languageSwitchLabel = currentLanguage === 'zh' ? t('cta.switchToEnglish') : t('cta.switchToChinese')

  const go = (path: string) => {
    navigate(path)
    setMobileMenuOpen(false)
  }

  const handleSwitchLanguage = () => {
    const nextLanguage: LandingLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(LANDING_LANGUAGE_STORAGE_KEY, nextLanguage)
    void i18n.changeLanguage(nextLanguage).finally(() => {
      window.location.reload()
    })
  }

  useEffect(() => {
    const handleScroll = () => {
      const scrollTop = window.scrollY || document.documentElement.scrollTop
      setScrolled(scrollTop > 50)
      setShowBackTop(scrollTop > 50)
    }
    window.addEventListener('scroll', handleScroll)
    handleScroll()
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  useEffect(() => {
    const loadPageAnnouncement = async () => {
      try {
        const res = await api.get('/api/public/settings')
        const enabled =
          res?.data?.home_page_announcement_enabled === true ||
          (res?.data?.home_page_announcement_enabled == null && res?.data?.page_announcement_enabled === true)
        const content = String(
          res?.data?.home_page_announcement_content ??
          res?.data?.page_announcement_content ??
          ''
        )
        setPageAnnouncementEnabled(enabled)
        setPageAnnouncementContent(content)
      } catch {
        setPageAnnouncementEnabled(false)
        setPageAnnouncementContent('')
      }
    }

    loadPageAnnouncement()
  }, [])

  useEffect(() => {
    if (!isMobile) {
      setMobileMenuOpen(false)
    }
  }, [isMobile])

  const featureIcons = [
    <RocketOutlined style={{ fontSize: 32, color: token.colorPrimary }} />,
    <SafetyOutlined style={{ fontSize: 32, color: token.colorSuccess }} />,
    <BarChartOutlined style={{ fontSize: 32, color: token.colorWarning }} />,
    <CloudOutlined style={{ fontSize: 32, color: token.colorInfo }} />,
    <ApiOutlined style={{ fontSize: 32, color: token.colorError }} />,
    <TeamOutlined style={{ fontSize: 32, color: token.colorPrimary }} />
  ]

  const featureItems = t('coreFeatures.items', { returnObjects: true }) as unknown as Array<{
    title: string
    description: string
  }>
  const features = featureItems.map((item, index) => ({
    icon: featureIcons[index],
    title: item.title,
    description: item.description
  }))
  const stats = t('stats', { returnObjects: true }) as unknown as Array<{ value: string; label: string }>
  const highlights = t('highlightsSection.items', { returnObjects: true }) as unknown as string[]

  return (
    <div style={{ minHeight: '100vh', background: '#fff', overflowX: 'hidden' }}>
      {/* 导航栏 */}
      <header
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          zIndex: 1000,
          background: scrolled ? 'rgba(255,255,255,0.95)' : 'transparent',
          backdropFilter: scrolled ? 'blur(10px)' : 'none',
          boxShadow: scrolled ? '0 2px 8px rgba(0,0,0,0.06)' : 'none',
          transition: 'all 0.3s ease'
        }}
      >
        <div
          style={{
            maxWidth: 1200,
            margin: '0 auto',
            padding: isMobile ? '12px 16px' : '16px 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between'
          }}
        >
          <Space size={isMobile ? 10 : 12}>
            <div
              style={{
                width: isMobile ? 32 : 36,
                height: isMobile ? 32 : 36,
                background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
                borderRadius: 8,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}
            >
              <RocketOutlined style={{ color: '#fff', fontSize: isMobile ? 18 : 20 }} />
            </div>
            <Text strong style={{ fontSize: isMobile ? 18 : 20, whiteSpace: 'nowrap' }}>SWM</Text>
            <Badge count="Beta" style={{ backgroundColor: '#52c41a', whiteSpace: 'nowrap' }} />
          </Space>
          {isMobile ? (
            <Button
              type="text"
              icon={<MenuOutlined />}
              aria-label={t('nav.menuAriaLabel')}
              onClick={() => setMobileMenuOpen(true)}
              style={{ fontSize: 18 }}
            />
          ) : (
            <Space size={24}>
              <Button type="text" onClick={() => go('/')}>
                {t('nav.home')}
              </Button>
              <Button type="text" onClick={() => go('/product')}>
                {t('nav.product')}
              </Button>
              <Button type="text" onClick={() => go('/pricing')}>
                {t('nav.pricing')}
              </Button>
              <Button type="text" onClick={() => go('/changelog')}>
                {t('nav.changelog')}
              </Button>
              <Button type="text" onClick={() => go('/api-docs')}>
                {t('nav.apiDocs')}
              </Button>
              <Button type="text" onClick={() => go('/login')}>
                {t('nav.login')}
              </Button>
              <Button type="primary" size="large" onClick={() => go('/register')}>
                {t('nav.freeTrial')}
              </Button>
              <Button
                size="large"
                icon={<GlobalOutlined />}
                onClick={handleSwitchLanguage}
              >
                {languageSwitchLabel}
              </Button>
            </Space>
          )}
        </div>
      </header>

      <Drawer
        title={t('nav.drawerTitle')}
        placement="right"
        open={mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
        width={280}
        styles={{ body: { padding: 16 } }}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Button block type="text" onClick={() => go('/')}>{t('nav.home')}</Button>
          <Button block type="text" onClick={() => go('/product')}>{t('nav.product')}</Button>
          <Button block type="text" onClick={() => go('/pricing')}>{t('nav.pricing')}</Button>
          <Button block type="text" onClick={() => go('/changelog')}>{t('nav.changelog')}</Button>
          <Button block type="text" onClick={() => go('/api-docs')}>{t('nav.apiDocs')}</Button>
          <Button block type="text" onClick={() => go('/login')}>{t('nav.login')}</Button>
          <Button block type="primary" onClick={() => go('/register')}>{t('nav.freeTrial')}</Button>
          <Button block icon={<GlobalOutlined />} onClick={handleSwitchLanguage}>
            {languageSwitchLabel}
          </Button>
        </Space>
      </Drawer>

      {/* Hero 区域 */}
      <section
        style={{
          padding: isMobile ? '104px 16px 56px' : '140px 24px 80px',
          background: 'linear-gradient(180deg, #f0f5ff 0%, #fff 100%)',
          textAlign: 'center'
        }}
      >
        <div style={{ maxWidth: 900, margin: '0 auto' }}>
          {pageAnnouncementEnabled && pageAnnouncementContent && (
            <Alert
              style={{ marginBottom: isMobile ? 16 : 24, textAlign: 'left' }}
              type="warning"
              showIcon
              message={<div style={{ whiteSpace: 'pre-wrap' }}>{pageAnnouncementContent}</div>}
            />
          )}
          <Title style={{ fontSize: isMobile ? 52 : 56, marginBottom: isMobile ? 16 : 24, lineHeight: 1.2 }}>
            {t('hero.titleLine1')}
            <br />
            <span style={{ color: token.colorPrimary }}>{t('hero.titleHighlight')}</span>
          </Title>
          <Paragraph style={{ fontSize: isMobile ? 16 : 20, color: '#666', marginBottom: isMobile ? 28 : 40 }}>
            {t('hero.descriptionLine1')}
            {!isMobile && <br />}
            {t('hero.descriptionLine2')}
          </Paragraph>
          <Space
            size={16}
            direction={isMobile ? 'vertical' : 'horizontal'}
            style={isMobile ? { width: '100%' } : undefined}
          >
            <Button
              type="primary"
              size="large"
              icon={<RocketOutlined />}
              onClick={() => go('/register')}
              style={isMobile ? { width: '100%', height: 48, fontSize: 16 } : { height: 48, padding: '0 32px', fontSize: 16 }}
            >
              {t('hero.startNow')}
            </Button>
            <Button
              size="large"
              icon={<GithubOutlined />}
              onClick={() => window.open('https://github.com/salokrwhite/swm-sdk', '_blank', 'noopener,noreferrer')}
              style={isMobile ? { width: '100%', height: 48, fontSize: 16 } : { height: 48, padding: '0 32px', fontSize: 16 }}
            >
              {t('hero.viewDocs')}
            </Button>
          </Space>

          {/* 统计数据 */}
          <Row
            gutter={isMobile ? [16, 20] : 48}
            style={{
              marginTop: isMobile ? 48 : 80,
              padding: isMobile ? '28px 0 0' : '40px 0',
              borderTop: '1px solid #e8e8e8'
            }}
          >
            {stats.map((stat, index) => (
              <Col xs={12} md={6} key={index}>
                <div style={{ textAlign: 'center' }}>
                  <Title level={isMobile ? 3 : 2} style={{ margin: 0, color: token.colorPrimary }}>
                    {stat.value}
                  </Title>
                  <Text type="secondary">{stat.label}</Text>
                </div>
              </Col>
            ))}
          </Row>
        </div>
      </section>

      {/* 功能特性 */}
      <section style={{ padding: isMobile ? '56px 16px' : '80px 24px', background: '#fff' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <div style={{ textAlign: 'center', marginBottom: isMobile ? 36 : 64 }}>
            <Title level={2}>{t('coreFeatures.title')}</Title>
            <Paragraph style={{ fontSize: 16, color: '#666' }}>
              {t('coreFeatures.subtitle')}
            </Paragraph>
          </div>
          <Row gutter={isMobile ? [16, 16] : [32, 32]}>
            {features.map((feature, index) => (
              <Col xs={24} sm={12} lg={8} key={index}>
                <Card
                  hoverable
                  style={{ height: '100%', borderRadius: 12 }}
                  styles={{ body: { padding: isMobile ? 20 : 32 } }}
                >
                  <div style={{ marginBottom: 16 }}>{feature.icon}</div>
                  <Title level={4} style={{ marginBottom: 12 }}>
                    {feature.title}
                  </Title>
                  <Paragraph type="secondary">{feature.description}</Paragraph>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      </section>

      {/* 亮点展示 */}
      <section style={{ padding: isMobile ? '56px 16px' : '80px 24px', background: '#f6ffed' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <Row gutter={isMobile ? [24, 24] : 64} align="middle">
            <Col xs={24} lg={12}>
              <Title level={2} style={{ marginBottom: 24 }}>
                {t('highlightsSection.title')}
              </Title>
              <Paragraph style={{ fontSize: 16, color: '#666', marginBottom: 32 }}>
                {t('highlightsSection.description')}
              </Paragraph>
              <Space direction="vertical" size={16} style={{ width: '100%' }}>
                {highlights.map((item, index) => (
                  <div key={index} style={{ display: 'flex', alignItems: 'center' }}>
                    <CheckCircleOutlined
                      style={{ color: token.colorSuccess, marginRight: 12, fontSize: 18 }}
                    />
                    <Text style={{ fontSize: 16 }}>{item}</Text>
                  </div>
                ))}
              </Space>
              <Button
                type="primary"
                size="large"
                icon={<ArrowRightOutlined />}
                style={isMobile ? { marginTop: 28, height: 44, width: '100%' } : { marginTop: 32, height: 44 }}
                onClick={() => go('/login')}
              >
                {t('highlightsSection.learnMore')}
              </Button>
            </Col>
          </Row>
        </div>
      </section>

      {/* 全球化部署 */}
      <section
        style={{
          padding: isMobile ? '56px 16px' : '80px 24px',
          background: 'linear-gradient(135deg, #27b6f0 0%, #1aa6ff 55%, #0f7ad8 100%)'
        }}
      >
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <Row gutter={isMobile ? [24, 24] : 64} align="middle" justify="center">
            <Col xs={24} md={16} lg={12}>
              <div style={{ padding: isMobile ? '0' : '24px 0', textAlign: 'center' }}>
                <GlobalOutlined style={{ fontSize: isMobile ? 32 : 40, color: '#fff', marginBottom: 16 }} />
                <Title level={3} style={{ color: '#fff', marginBottom: 16 }}>
                  {t('globalSection.title')}
                </Title>
                <Paragraph style={{ fontSize: 16, color: 'rgba(255,255,255,0.92)', marginBottom: 32 }}>
                  {t('globalSection.description')}
                </Paragraph>
                <Row gutter={[16, 16]}>
                  <Col xs={8}>
                    <div style={{ textAlign: 'center' }}>
                      <Title level={3} style={{ color: '#fff', margin: 0 }}>30+</Title>
                      <Text style={{ color: 'rgba(255,255,255,0.8)' }}>{t('globalSection.nodes')}</Text>
                    </div>
                  </Col>
                  <Col xs={8}>
                    <div style={{ textAlign: 'center' }}>
                      <Title level={3} style={{ color: '#fff', margin: 0 }}>5</Title>
                      <Text style={{ color: 'rgba(255,255,255,0.8)' }}>{t('globalSection.continents')}</Text>
                    </div>
                  </Col>
                  <Col xs={8}>
                    <div style={{ textAlign: 'center' }}>
                      <Title level={3} style={{ color: '#fff', margin: 0 }}>&lt;100ms</Title>
                      <Text style={{ color: 'rgba(255,255,255,0.8)' }}>{t('globalSection.latency')}</Text>
                    </div>
                  </Col>
                </Row>
              </div>
            </Col>
          </Row>
        </div>
      </section>

      {/* 套餐方案 */}
      <section style={{ padding: isMobile ? '56px 16px' : '90px 24px', background: '#f7f9fc' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <div style={{ textAlign: 'center', marginBottom: isMobile ? 36 : 64 }}>
            <Title level={2}>{t('pricing.title')}</Title>
            <Paragraph style={{ fontSize: 16, color: '#666' }}>
              {t('pricing.subtitle')}
            </Paragraph>
          </div>
          <Row gutter={isMobile ? [16, 16] : [32, 32]}>
            <Col xs={24} md={8}>
              <Card
                hoverable
                style={{
                  height: '100%',
                  borderRadius: 16,
                  border: '2px solid #e6f4ff',
                  boxShadow: 'none'
                }}
                styles={{ body: { padding: isMobile ? 20 : 32 } }}
              >
                <Space direction="vertical" size={16} style={{ width: '100%' }}>
                  <div>
                    <Title level={3} style={{ marginTop: 0, marginBottom: 8 }}>Free</Title>
                  </div>
                  <Space direction="vertical" size={12}>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.free.feature1')}</Text>
                    </Space>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.free.feature2')}</Text>
                    </Space>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.free.feature3')}</Text>
                    </Space>
                  </Space>
                  <Button type="primary" onClick={() => go('/register')} style={isMobile ? { width: '100%' } : undefined}>
                    {t('pricing.free.cta')}
                  </Button>
                </Space>
              </Card>
            </Col>
            <Col xs={24} md={8}>
              <Card
                hoverable
                style={{
                  height: '100%',
                  borderRadius: 16,
                  border: '2px solid #d9f7be',
                  boxShadow: 'none'
                }}
                styles={{ body: { padding: isMobile ? 20 : 32 } }}
              >
                <Space direction="vertical" size={16} style={{ width: '100%' }}>
                  <div>
                    <Title level={3} style={{ marginTop: 0, marginBottom: 8 }}>Team</Title>
                  </div>
                  <Space direction="vertical" size={12}>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.team.feature1')}</Text>
                    </Space>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.team.feature2')}</Text>
                    </Space>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.team.feature3')}</Text>
                    </Space>
                  </Space>
                  <Button type="primary" onClick={() => go('/enterprise-register')} style={isMobile ? { width: '100%' } : undefined}>
                    {t('pricing.team.cta')}
                  </Button>
                </Space>
              </Card>
            </Col>
            <Col xs={24} md={8}>
              <Card
                hoverable
                style={{
                  height: '100%',
                  borderRadius: 16,
                  border: '2px solid #ffd591',
                  boxShadow: 'none'
                }}
                styles={{ body: { padding: isMobile ? 20 : 32 } }}
              >
                <Space direction="vertical" size={16} style={{ width: '100%' }}>
                  <div>
                    <Title level={3} style={{ marginTop: 0, marginBottom: 8 }}>Enterprise</Title>
                  </div>
                  <Space direction="vertical" size={12}>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.enterprise.feature1')}</Text>
                    </Space>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.enterprise.feature2')}</Text>
                    </Space>
                    <Space align="center" size={12}>
                      <CheckCircleOutlined style={{ color: token.colorSuccess, fontSize: 18 }} />
                      <Text>{t('pricing.enterprise.feature3')}</Text>
                    </Space>
                  </Space>
                  <Button onClick={() => go('/enterprise-register')} style={isMobile ? { width: '100%' } : undefined}>
                    {t('pricing.enterprise.cta')}
                  </Button>
                </Space>
              </Card>
            </Col>
          </Row>
        </div>
      </section>

      {/* CTA 区域 */}
      <section style={{ padding: isMobile ? '64px 16px' : '100px 24px', background: '#fff', textAlign: 'center' }}>
        <div style={{ maxWidth: 600, margin: '0 auto' }}>
          <Title level={2} style={{ marginBottom: 24 }}>
            {t('cta.title')}
          </Title>
          <Paragraph style={{ fontSize: 16, color: '#666', marginBottom: 40 }}>
            {t('cta.description')}
          </Paragraph>
          <Space size={16} direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Button
              type="primary"
              size="large"
              onClick={() => go('/login')}
              style={isMobile ? { width: '100%', height: 48, fontSize: 16 } : { height: 48, padding: '0 40px', fontSize: 16 }}
            >
              {t('cta.start')}
            </Button>
            <Button
              size="large"
              style={isMobile ? { width: '100%', height: 48, fontSize: 16 } : { height: 48, padding: '0 40px', fontSize: 16 }}
            >
              {t('cta.contactSales')}
            </Button>
          </Space>
        </div>
      </section>

      {/* 页脚 */}
      <footer style={{ padding: isMobile ? '40px 16px 28px' : '60px 24px 40px', background: '#001529', color: '#fff' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <Row gutter={isMobile ? [16, 24] : 48}>
            <Col xs={24} md={8}>
              <Space style={{ marginBottom: 16 }}>
                <div
                  style={{
                    width: 32,
                    height: 32,
                    background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
                    borderRadius: 6,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}
                >
                  <RocketOutlined style={{ color: '#fff', fontSize: 16 }} />
                </div>
                <Text strong style={{ color: '#fff', fontSize: 18 }}>SWM</Text>
              </Space>
              <Paragraph style={{ color: 'rgba(255,255,255,0.65)' }}>
                {t('footer.description')}
              </Paragraph>
            </Col>
            <Col xs={24} md={8}>
              <Title level={5} style={{ color: '#fff', marginBottom: 16 }}>{t('footer.product')}</Title>
              <Space direction="vertical">
                <Text style={{ color: 'rgba(255,255,255,0.65)', cursor: 'pointer' }} onClick={() => go('/product')}>{t('nav.product')}</Text>
                <Text style={{ color: 'rgba(255,255,255,0.65)', cursor: 'pointer' }} onClick={() => go('/pricing')}>{t('nav.pricing')}</Text>
                <Text style={{ color: 'rgba(255,255,255,0.65)', cursor: 'pointer' }} onClick={() => go('/changelog')}>{t('nav.changelog')}</Text>
                <Text style={{ color: 'rgba(255,255,255,0.65)', cursor: 'pointer' }} onClick={() => go('/api-docs')}>{t('nav.apiDocs')}</Text>
              </Space>
            </Col>
            <Col xs={24} md={8}>
              <Title level={5} style={{ color: '#fff', marginBottom: 16 }}>{t('footer.support')}</Title>
              <Space direction="vertical">
                <a
                  href="mailto:report@service.anteasy.com"
                  style={{ color: 'rgba(255,255,255,0.65)' }}
                >
                  {t('footer.contactUs')}
                </a>
                <Text style={{ color: 'rgba(255,255,255,0.65)', cursor: 'pointer' }} onClick={() => go('/service-status')}>
                  {t('footer.serviceStatus')}
                </Text>
              </Space>
            </Col>
          </Row>
          <div
            style={{
              marginTop: isMobile ? 28 : 40,
              paddingTop: 24,
              borderTop: '1px solid rgba(255,255,255,0.1)',
              textAlign: 'center',
              color: 'rgba(255,255,255,0.45)',
              fontSize: isMobile ? 12 : 14
            }}
          >
            {t('footer.copyright')}
          </div>
        </div>
      </footer>
      {showBackTop && (
        <Button
          type="primary"
          shape="circle"
          icon={<ArrowUpOutlined />}
          onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
          style={{
            position: 'fixed',
            right: isMobile ? 16 : 32,
            bottom: isMobile ? 24 : 40,
            width: 48,
            height: 48,
            boxShadow: '0 12px 24px rgba(0,0,0,0.2)',
            zIndex: 1200
          }}
        />
      )}
    </div>
  )
}
