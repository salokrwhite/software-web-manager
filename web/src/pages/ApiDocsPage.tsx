import { ApiOutlined, ArrowUpOutlined, GlobalOutlined, MenuOutlined, RocketOutlined } from '@ant-design/icons'
import { Badge, Button, Col, Drawer, Grid, Row, Space, Typography } from 'antd'
import i18next from 'i18next'
import { useEffect, useState } from 'react'
import { initReactI18next, useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import Docs from './Docs'

const { Title, Paragraph, Text } = Typography

type ApiDocsLanguage = 'zh' | 'en'

const API_DOCS_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialApiDocsLanguage = (): ApiDocsLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(API_DOCS_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const apiDocsResources = {
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
      apiDocsPage: {
        title: 'API 文档',
        subtitle: '官网公开完整文档页面，包含接口说明、SDK 示例与接入步骤。'
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
      apiDocsPage: {
        title: 'API Documentation',
        subtitle: 'Public documentation page with API specs, SDK examples, and integration steps.'
      }
    }
  }
} as const

if (!i18next.isInitialized) {
  void i18next.use(initReactI18next).init({
    resources: apiDocsResources,
    lng: getInitialApiDocsLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', apiDocsResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', apiDocsResources.en.translation, true, true)
}

export default function ApiDocsPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [scrolled, setScrolled] = useState(false)
  const [showBackTop, setShowBackTop] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const currentLanguage: ApiDocsLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const languageSwitchLabel = currentLanguage === 'zh' ? t('cta.switchToEnglish') : t('cta.switchToChinese')

  const go = (path: string) => {
    navigate(path)
    setMobileMenuOpen(false)
  }

  const handleSwitchLanguage = () => {
    const nextLanguage: ApiDocsLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(API_DOCS_LANGUAGE_STORAGE_KEY, nextLanguage)
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
          padding: isMobile ? '104px 16px 36px' : '140px 24px 56px',
          background: 'linear-gradient(180deg, #f0f5ff 0%, #fff 100%)'
        }}
      >
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <Space align="center" size={isMobile ? 10 : 12} style={{ marginBottom: isMobile ? 10 : 12 }}>
            <ApiOutlined style={{ fontSize: isMobile ? 24 : 28, color: '#1677ff' }} />
            <Title style={{ margin: 0, fontSize: isMobile ? 34 : 44, lineHeight: 1.2 }}>{t('apiDocsPage.title')}</Title>
          </Space>
          <Paragraph style={{ fontSize: isMobile ? 15 : 16, color: '#666', marginBottom: 0 }}>
            {t('apiDocsPage.subtitle')}
          </Paragraph>
        </div>
      </section>

      <section style={{ padding: isMobile ? '0 16px 56px' : '0 24px 80px', background: '#fff' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          <div style={isMobile ? { overflowX: 'auto' } : undefined}>
            <Docs />
          </div>
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
