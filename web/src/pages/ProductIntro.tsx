import { useEffect, useState } from 'react'
import {
  ApiOutlined,
  ArrowUpOutlined,
  AuditOutlined,
  BarChartOutlined,
  CheckCircleOutlined,
  CloudUploadOutlined,
  DeploymentUnitOutlined,
  GlobalOutlined,
  MenuOutlined,
  RocketOutlined,
  SafetyOutlined,
  TeamOutlined
} from '@ant-design/icons'
import { Badge, Button, Card, Col, Drawer, Grid, Row, Space, Typography, theme } from 'antd'
import i18next from 'i18next'
import { initReactI18next, useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

const { Title, Paragraph, Text } = Typography
const { useToken } = theme

type ProductLanguage = 'zh' | 'en'

const PRODUCT_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialProductLanguage = (): ProductLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(PRODUCT_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const productResources = {
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
      cta: {
        switchToEnglish: 'English',
        switchToChinese: '中文'
      },
      footer: {
        description: '企业级软件版本管理平台，让版本管理更简单、更安全、更智能。',
        product: '产品',
        support: '支持',
        contactUs: '联系我们',
        serviceStatus: '服务状态',
        copyright: '© 1996-2024 小白科学研究院 All rights reserved. 吉ICP备2025026240号-3'
      },
      productIntro: {
        heroTitle: 'SWM 产品能力全景',
        heroHighlight: '覆盖版本发布全生命周期',
        heroDescription1: '从应用管理、版本发布、灰度策略到设备在线与数据分析，再到组织协作、工单审计与系统设置，',
        heroDescription2: '一个系统完成研发发布治理闭环。',
        coreTitle: '核心模块',
        coreSubtitle: '以下能力均已在系统中落地，可直接用于企业软件版本管理场景。',
        modules: [
          {
            title: '应用与版本发布',
            desc: '覆盖应用创建、版本管理、发布计划、模板复用、审核与回滚全流程。'
          },
          {
            title: '灰度与区域策略',
            desc: '支持渠道灰度、白名单设备、区域规则与投放窗口，降低发布风险。'
          },
          {
            title: '数据分析',
            desc: '支持事件总览、版本分布、失败分析与手动刷新聚合，定位发布效果。'
          },
          {
            title: '设备与在线监控',
            desc: '管理设备历史、实时在线状态、心跳周期与设备过滤筛选。'
          },
          {
            title: '组织协作',
            desc: '支持个人与企业空间、成员管理、加入申请审批与组织切换。'
          },
          {
            title: '审计与工单',
            desc: '提供审计日志留痕、工单提交流转与系统级问题跟踪。'
          },
          {
            title: 'SDK 与开放能力',
            desc: '提供 App Secret 和多语言 SDK，支持更新检查、事件、心跳、反馈接入。'
          }
        ],
        scenariosTitle: '适用场景',
        scenarios: [
          '桌面客户端、企业内部工具、渠道分发应用持续交付。',
          '需要审批流、审计留痕与可回滚发布的研发团队。',
          '需要组织隔离、角色权限、系统级设置统一治理的企业环境。'
        ],
        finalTitle: '准备开始管理你的版本发布了吗？',
        finalDescription: '使用 SWM 建立统一的发布流程、权限治理与数据观测体系。',
        finalPrimary: '免费开始使用',
        finalSecondary: '返回首页'
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
      cta: {
        switchToEnglish: 'English',
        switchToChinese: '中文'
      },
      footer: {
        description: 'Enterprise software version management platform: simpler, safer, smarter.',
        product: 'Product',
        support: 'Support',
        contactUs: 'Contact Us',
        serviceStatus: 'Service Status',
        copyright: '© 1996-2024 Xiaobai Institute. All rights reserved. ICP 吉ICP备2025026240号-3'
      },
      productIntro: {
        heroTitle: 'SWM Product Overview',
        heroHighlight: 'Complete Release Lifecycle Coverage',
        heroDescription1: 'From app management, release orchestration, and canary strategy to device monitoring and analytics, plus team collaboration, ticketing, and system settings,',
        heroDescription2: 'one platform closes the loop for software release governance.',
        coreTitle: 'Core Modules',
        coreSubtitle: 'These capabilities are production-ready and can be used directly in enterprise release management scenarios.',
        modules: [
          {
            title: 'App and Release Management',
            desc: 'Covers app creation, version management, release planning, template reuse, approvals, and rollback.'
          },
          {
            title: 'Canary and Region Strategy',
            desc: 'Supports channel canary, device allowlist, region rules, and rollout windows to reduce release risk.'
          },
          {
            title: 'Analytics',
            desc: 'Provides event overview, version distribution, failure analysis, and manual refresh aggregation.'
          },
          {
            title: 'Device and Online Monitoring',
            desc: 'Manages device history, real-time online status, heartbeat interval, and device filtering.'
          },
          {
            title: 'Organization Collaboration',
            desc: 'Supports personal and enterprise spaces, member management, join approvals, and org switching.'
          },
          {
            title: 'Audit and Ticketing',
            desc: 'Provides audit trail records, ticket workflow, and system-level issue tracking.'
          },
          {
            title: 'SDK and Open Capabilities',
            desc: 'Provides App Secret and multi-language SDKs for update checks, events, heartbeat, and feedback.'
          }
        ],
        scenariosTitle: 'Use Cases',
        scenarios: [
          'Continuous delivery for desktop clients, internal enterprise tools, and channel distribution apps.',
          'Engineering teams requiring approvals, audit trails, and rollback-ready releases.',
          'Enterprise environments requiring org isolation, RBAC, and centralized system governance.'
        ],
        finalTitle: 'Ready to manage your releases?',
        finalDescription: 'Use SWM to build unified release workflow, permission governance, and observability.',
        finalPrimary: 'Start for Free',
        finalSecondary: 'Back to Home'
      }
    }
  }
} as const

if (!i18next.isInitialized) {
  void i18next.use(initReactI18next).init({
    resources: productResources,
    lng: getInitialProductLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', productResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', productResources.en.translation, true, true)
}

export default function ProductIntro() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { token } = useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [scrolled, setScrolled] = useState(false)
  const [showBackTop, setShowBackTop] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const currentLanguage: ProductLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const languageSwitchLabel = currentLanguage === 'zh' ? t('cta.switchToEnglish') : t('cta.switchToChinese')

  const go = (path: string) => {
    navigate(path)
    setMobileMenuOpen(false)
  }

  const handleSwitchLanguage = () => {
    const nextLanguage: ProductLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(PRODUCT_LANGUAGE_STORAGE_KEY, nextLanguage)
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
    if (!isMobile) {
      setMobileMenuOpen(false)
    }
  }, [isMobile])

  const moduleIcons = [
    <CloudUploadOutlined style={{ fontSize: 26 }} />,
    <SafetyOutlined style={{ fontSize: 26 }} />,
    <BarChartOutlined style={{ fontSize: 26 }} />,
    <DeploymentUnitOutlined style={{ fontSize: 26 }} />,
    <TeamOutlined style={{ fontSize: 26 }} />,
    <AuditOutlined style={{ fontSize: 26 }} />,
    <ApiOutlined style={{ fontSize: 26 }} />
  ]

  const moduleItems = t('productIntro.modules', { returnObjects: true }) as unknown as Array<{
    title: string
    desc: string
  }>
  const modules = moduleItems.map((item, index) => ({
    icon: moduleIcons[index] ?? <ApiOutlined style={{ fontSize: 26 }} />,
    title: item.title,
    desc: item.desc
  }))
  const scenarios = t('productIntro.scenarios', { returnObjects: true }) as unknown as string[]

  return (
    <div style={{ minHeight: '100vh', background: '#fff', overflowX: 'hidden' }}>
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
              <Button type="text" onClick={() => go('/')}>{t('nav.home')}</Button>
              <Button type="text" onClick={() => go('/product')}>{t('nav.product')}</Button>
              <Button type="text" onClick={() => go('/pricing')}>{t('nav.pricing')}</Button>
              <Button type="text" onClick={() => go('/changelog')}>{t('nav.changelog')}</Button>
              <Button type="text" onClick={() => go('/api-docs')}>{t('nav.apiDocs')}</Button>
              <Button type="text" onClick={() => go('/login')}>{t('nav.login')}</Button>
              <Button type="primary" size="large" onClick={() => go('/register')}>{t('nav.freeTrial')}</Button>
              <Button size="large" icon={<GlobalOutlined />} onClick={handleSwitchLanguage}>
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

      <section
        style={{
          padding: isMobile ? '104px 16px 56px' : '140px 24px 80px',
          background: 'linear-gradient(180deg, #f0f5ff 0%, #fff 100%)',
          textAlign: 'center'
        }}
      >
        <div style={{ maxWidth: 980, margin: '0 auto' }}>
          <Title style={{ fontSize: isMobile ? 38 : 52, marginBottom: isMobile ? 16 : 18, lineHeight: 1.2 }}>
            {t('productIntro.heroTitle')}
            <br />
            <span style={{ color: token.colorPrimary }}>{t('productIntro.heroHighlight')}</span>
          </Title>
          <Paragraph style={{ fontSize: isMobile ? 16 : 18, color: '#666', marginBottom: isMobile ? 16 : 24 }}>
            {t('productIntro.heroDescription1')}
            {!isMobile && <br />}
            {t('productIntro.heroDescription2')}
          </Paragraph>
        </div>
      </section>

      <section style={{ padding: isMobile ? '56px 16px' : '80px 24px', background: '#fff' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <div style={{ textAlign: 'center', marginBottom: isMobile ? 36 : 56 }}>
            <Title level={2}>{t('productIntro.coreTitle')}</Title>
            <Paragraph style={{ fontSize: 16, color: '#666' }}>
              {t('productIntro.coreSubtitle')}
            </Paragraph>
          </div>

          <Row gutter={isMobile ? [16, 16] : [24, 24]}>
            {modules.map((item, index) => (
              <Col key={`${item.title}-${index}`} xs={24} sm={12} lg={8}>
                <Card
                  hoverable
                  style={{ borderRadius: 12, height: '100%' }}
                  styles={{ body: { padding: isMobile ? 20 : 28 } }}
                >
                  <Space direction="vertical" size={10}>
                    <div>{item.icon}</div>
                    <Text strong style={{ fontSize: 16 }}>{item.title}</Text>
                    <Text type="secondary">{item.desc}</Text>
                  </Space>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      </section>

      <section style={{ padding: isMobile ? '56px 16px' : '72px 24px', background: '#f6ffed' }}>
        <div style={{ maxWidth: 1080, margin: '0 auto' }}>
          <Title level={2} style={{ marginBottom: isMobile ? 16 : 20 }}>{t('productIntro.scenariosTitle')}</Title>
          <Space direction="vertical" size={isMobile ? 12 : 14} style={{ width: '100%' }}>
            {scenarios.map((item, index) => (
              <Text key={`${item}-${index}`} style={{ fontSize: isMobile ? 15 : 16 }}>
                <CheckCircleOutlined style={{ color: token.colorSuccess, marginRight: 10 }} />
                {item}
              </Text>
            ))}
          </Space>
        </div>
      </section>

      <section style={{ padding: isMobile ? '64px 16px' : '88px 24px', background: '#fff', textAlign: 'center' }}>
        <div style={{ maxWidth: 680, margin: '0 auto' }}>
          <Title level={2} style={{ marginBottom: isMobile ? 12 : 16 }}>{t('productIntro.finalTitle')}</Title>
          <Paragraph style={{ fontSize: isMobile ? 15 : 16, color: '#666', marginBottom: isMobile ? 24 : 28 }}>
            {t('productIntro.finalDescription')}
          </Paragraph>
          <Space size={12} direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Button
              type="primary"
              size="large"
              onClick={() => go('/register')}
              style={isMobile ? { width: '100%', height: 46, fontSize: 16 } : undefined}
            >
              {t('productIntro.finalPrimary')}
            </Button>
            <Button
              size="large"
              onClick={() => go('/')}
              style={isMobile ? { width: '100%', height: 46, fontSize: 16 } : undefined}
            >
              {t('productIntro.finalSecondary')}
            </Button>
          </Space>
        </div>
      </section>

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
