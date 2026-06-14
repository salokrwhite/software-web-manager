import {
  Button,
  Card,
  Checkbox,
  Col,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message
} from 'antd'
import { useState } from 'react'
import api from '../../../api/client'
import ReleasePolicySummaryCard from '../components/ReleasePolicySummaryCard'
import FeatureGuide, { GuideTag } from '../components/FeatureGuide'
import { parseRegionRules } from '../utils/parse'
import {
  applyRegionTemplateToForm,
  buildRegionRulesFromTemplates,
  buildRegionTemplate,
  filterCityOptions,
  filterRegionOptions,
  formatCityLabel,
  formatRegionLabel,
  hasRegionRules
} from '../utils/region'

const { Text } = Typography

type RegionRulesTabProps = {
  appId: string
  releaseChannels: any[]
  regionTemplates: any[]
  activeRegionTemplateId: string
  regionEnabled: boolean
  regionOptions: { countries: string[]; provinces: string[]; cities: string[] }
  isLocked: boolean
  reload: () => void
  setRegionTemplates: (templates: any[]) => void
  setActiveRegionTemplateId: (id: string) => void
  setRegionEnabled: (enabled: boolean) => void
}

export default function RegionRulesTab({
  appId,
  releaseChannels,
  regionTemplates,
  activeRegionTemplateId,
  regionEnabled,
  regionOptions,
  isLocked,
  reload,
  setRegionTemplates,
  setActiveRegionTemplateId,
  setRegionEnabled
}: RegionRulesTabProps) {
  const [regionChannelOpen, setRegionChannelOpen] = useState(false)
  const [editingRegionChannel, setEditingRegionChannel] = useState<any>(null)
  const [regionLookupIP, setRegionLookupIP] = useState('')
  const [regionLookupResult, setRegionLookupResult] = useState<any>(null)
  const [regionLookupLoading, setRegionLookupLoading] = useState(false)
  const [regionTemplateOpen, setRegionTemplateOpen] = useState(false)
  const [editingRegionTemplate, setEditingRegionTemplate] = useState<any>(null)

  const [regionForm] = Form.useForm()
  const [regionChannelForm] = Form.useForm()
  const [regionTemplateForm] = Form.useForm()

  const emptyRegionRulesPayload = {
    mode: 'allow_deny',
    allow: { countries: [], provinces: [], cities: [] },
    deny: { countries: [], provinces: [], cities: [] }
  }

  const saveAppRegionRules = async (nextTemplates?: any[], nextActiveId?: string) => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const templates = nextTemplates ?? regionTemplates
      const activeId = nextActiveId ?? activeRegionTemplateId
      const payload = regionEnabled ? buildRegionRulesFromTemplates(templates, activeId) : null
      await api.patch(`/api/apps/${appId}/region-rules`, { region_rules: payload })
      message.success('区域规则已更新')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    }
  }

  const openRegionTemplateModal = (tpl?: any) => {
    setEditingRegionTemplate(tpl || null)
    regionTemplateForm.resetFields()
    applyRegionTemplateToForm(regionTemplateForm, tpl || {})
    setRegionTemplateOpen(true)
  }

  const saveRegionTemplate = async () => {
    try {
      const values = await regionTemplateForm.validateFields()
      const nextTemplate = buildRegionTemplate(values, editingRegionTemplate)
      let nextTemplates = [...regionTemplates]
      const idx = nextTemplates.findIndex((t) => t.id === nextTemplate.id)
      if (idx >= 0) {
        nextTemplates[idx] = nextTemplate
      } else {
        nextTemplates.push(nextTemplate)
      }
      const nextActiveId = activeRegionTemplateId || nextTemplate.id
      setRegionTemplates(nextTemplates)
      setActiveRegionTemplateId(nextActiveId)
      setRegionTemplateOpen(false)
      setEditingRegionTemplate(null)
      await saveAppRegionRules(nextTemplates, nextActiveId)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    }
  }

  const deleteRegionTemplate = async (tpl: any) => {
    const nextTemplates = regionTemplates.filter((t) => t.id !== tpl.id)
    const nextActiveId = nextTemplates.length > 0 ? (activeRegionTemplateId === tpl.id ? nextTemplates[0].id : activeRegionTemplateId) : ''
    setRegionTemplates(nextTemplates)
    setActiveRegionTemplateId(nextActiveId)
    await saveAppRegionRules(nextTemplates, nextActiveId)
  }

  const openRegionChannel = (record: any) => {
    setEditingRegionChannel(record)
    const rules = parseRegionRules(record.region_rules)
    const hasRules = hasRegionRules(rules)
    const activeId = rules?.active_template_id || (rules?.templates?.[0]?.id || '')
    const selectedTemplate = (rules?.templates || []).find((t: any) => t?.id === activeId) || rules?.templates?.[0]
    const allow = selectedTemplate?.allow || rules?.allow || {}
    const deny = selectedTemplate?.deny || rules?.deny || {}
    regionChannelForm.resetFields()
    regionChannelForm.setFieldsValue({
      inherit: !hasRules,
      template_id: activeId || '',
      template_name: selectedTemplate?.name || '',
      allow_countries: allow?.countries || [],
      allow_provinces: allow?.provinces || [],
      allow_cities: allow?.cities || [],
      deny_countries: deny?.countries || [],
      deny_provinces: deny?.provinces || [],
      deny_cities: deny?.cities || []
    })
    setRegionChannelOpen(true)
  }

  const saveRegionChannel = async () => {
    if (!editingRegionChannel) return
    try {
      const values = await regionChannelForm.validateFields()
      const inherit = !!values.inherit
      let payload: any = emptyRegionRulesPayload
      if (!inherit) {
        const templateId = values.template_id || ''
        const selected = regionTemplates.find((t) => t.id === templateId)
        if (selected) {
          payload = buildRegionRulesFromTemplates([selected], selected.id)
        } else {
          const customTemplate = buildRegionTemplate({
            name: values.template_name || '通道模板',
            allow_countries: values.allow_countries || [],
            allow_provinces: values.allow_provinces || [],
            allow_cities: values.allow_cities || [],
            deny_countries: values.deny_countries || [],
            deny_provinces: values.deny_provinces || [],
            deny_cities: values.deny_cities || []
          })
          payload = buildRegionRulesFromTemplates([customTemplate], customTemplate.id)
        }
      }
      await api.patch(`/api/release-channels/${editingRegionChannel.id}`, { region_rules: payload })
      message.success('通道区域规则已更新')
      setRegionChannelOpen(false)
      setEditingRegionChannel(null)
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    }
  }

  const resolveRegion = async () => {
    const ip = regionLookupIP.trim()
    if (!ip) {
      message.warning('请输入 IP')
      return
    }
    setRegionLookupLoading(true)
    try {
      const res = await api.get('/api/geo/resolve', { params: { ip } })
      setRegionLookupResult(res.data)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '解析失败')
    } finally {
      setRegionLookupLoading(false)
    }
  }

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24}>
        <FeatureGuide
          storageKey="region-rules"
          title="地区策略"
          summary={
            <>
              地区策略让你<Text strong>按用户所在的国家 / 省 / 城市来决定谁能收到更新</Text>。
              用「白名单」表示只发给指定地区，用「黑名单」表示屏蔽某些地区。
              不配置时所有地区都可以正常更新。
            </>
          }
          steps={[
            {
              title: '打开开关',
              description: <>把「应用级区域规则」右上角的<GuideTag>启用</GuideTag>打开，关闭则不做任何地区限制。</>
            },
            {
              title: '新建一个地区模板',
              description: <>点<GuideTag>新建模板</GuideTag>，在白名单/黑名单里选择国家、省、城市。把常用的地区组合存成模板，方便重复使用。</>
            },
            {
              title: '选择生效模板',
              description: <>在「当前生效模板」里选中刚才建好的模板，规则就会对整个应用生效。</>
            },
            {
              title: '（可选）单独覆盖某个通道',
              description: <>如果某个渠道需要不同的地区规则，在「通道覆盖规则」里点<GuideTag>设置</GuideTag>，不设置则默认继承应用级规则。</>
            }
          ]}
          tips={[
            <>地区填写格式为「国家·省·城市」逐级细化，例如国家填 <Text code>CN</Text>、省填 <Text code>CN|广东</Text>、城市填 <Text code>CN|广东|深圳</Text>。先选国家后，省和城市的下拉会自动给出可选项。</>,
            <>不确定某个用户 IP 属于哪个地区？用页面底部的「IP 区域解析」输入 IP 即可查询。</>,
            <>白名单和黑名单同时存在时，黑名单优先级更高（先满足白名单、再排除黑名单）。</>
          ]}
        />
      </Col>
      <Col xs={24}>
        <ReleasePolicySummaryCard
          releaseChannels={releaseChannels}
          title="发布策略摘要（人群视图）"
        />
      </Col>
      <Col xs={24}>
        <Card
          title="应用级区域规则"
          style={{ borderRadius: 12 }}
          extra={(
            <Space>
              <Text type="secondary">启用</Text>
              <Switch
                checked={regionEnabled}
                onChange={(checked) => {
                  setRegionEnabled(checked)
                  if (!checked) {
                    saveAppRegionRules([])
                  }
                }}
                disabled={isLocked}
              />
            </Space>
          )}
        >
          <Row gutter={[16, 16]}>
            <Col xs={24}>
              <Card size="small" title="模板列表" style={{ borderRadius: 8 }}>
                <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
                  <Col>
                    <Text type="secondary">共 {regionTemplates.length} 个模板</Text>
                  </Col>
                  <Col>
                    <Button type="primary" onClick={() => openRegionTemplateModal()} disabled={isLocked || !regionEnabled}>
                      新建模板
                    </Button>
                  </Col>
                </Row>
                <Table
                  rowKey="id"
                  dataSource={regionTemplates}
                  pagination={false}
                  locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无数据" /> }}
                  columns={[
                    { title: '名称', dataIndex: 'name' },
                    {
                      title: '操作',
                      render: (_: any, record: any) => (
                        <Space>
                          <Button size="small" onClick={() => openRegionTemplateModal(record)} disabled={isLocked}>编辑</Button>
                          <Button size="small" danger onClick={() => deleteRegionTemplate(record)} disabled={isLocked}>删除</Button>
                        </Space>
                      )
                    }
                  ]}
                />
              </Card>
            </Col>
            <Col xs={24}>
              <Card size="small" title="当前生效模板" style={{ borderRadius: 8 }}>
                <Form layout="vertical" form={regionForm}>
                  <Form.Item label="选择模板">
                    <Select
                      placeholder="请选择模板"
                      value={activeRegionTemplateId || undefined}
                      onChange={(value) => {
                        setActiveRegionTemplateId(value)
                        saveAppRegionRules(regionTemplates, value)
                      }}
                      options={regionTemplates.map((t) => ({ label: t.name, value: t.id }))}
                      notFoundContent="无数据"
                      disabled={!regionEnabled || isLocked}
                    />
                  </Form.Item>
                </Form>
                {!regionEnabled && (
                  <Text type="secondary">关闭后将不限制区域</Text>
                )}
              </Card>
            </Col>
          </Row>
        </Card>
      </Col>

      <Col xs={24}>
        <Card title="通道覆盖规则" style={{ borderRadius: 12 }}>
          <Table
            rowKey="id"
            dataSource={releaseChannels}
            pagination={{ pageSize: 5 }}
            locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无数据" /> }}
            columns={[
              { title: '渠道', dataIndex: 'channel_code' },
              { title: '版本', dataIndex: 'release_version' },
              { title: '状态', dataIndex: 'status' },
              {
                title: '规则',
                render: (_: any, record: any) => {
                  const hasRules = hasRegionRules(record.region_rules)
                  return hasRules ? <Tag color="blue">自定义</Tag> : <Tag>继承应用规则</Tag>
                }
              },
              {
                title: '操作',
                render: (_: any, record: any) => (
                  <Button size="small" onClick={() => openRegionChannel(record)} disabled={isLocked}>
                    设置
                  </Button>
                )
              }
            ]}
          />
        </Card>
      </Col>

      <Col xs={24}>
        <Card title="IP 区域解析" style={{ borderRadius: 12 }}>
          <Space>
            <Input
              placeholder="输入 IP 地址"
              value={regionLookupIP}
              onChange={(e) => setRegionLookupIP(e.target.value)}
            />
            <Button onClick={resolveRegion} loading={regionLookupLoading}>
              解析
            </Button>
          </Space>
          {regionLookupResult && (
            <Space style={{ display: 'flex', marginTop: 16, flexWrap: 'wrap' }}>
              <Tag>ISO: {regionLookupResult.iso || '-'}</Tag>
              <Tag>国家: {regionLookupResult.country || '-'}</Tag>
              <Tag>省/州: {regionLookupResult.province || '-'}</Tag>
              <Tag>城市: {regionLookupResult.city || '-'}</Tag>
            </Space>
          )}
        </Card>
      </Col>

      <Modal
        title="通道区域规则"
        open={regionChannelOpen}
        onOk={saveRegionChannel}
        onCancel={() => { setRegionChannelOpen(false) }}
        afterOpenChange={(open) => {
          if (!open) {
            setEditingRegionChannel(null)
            regionChannelForm.resetFields()
          }
        }}
        width={640}
      >
        <Form layout="vertical" form={regionChannelForm} style={{ marginTop: 16 }}>
          <Form.Item name="inherit" valuePropName="checked">
            <Checkbox>继承应用级规则</Checkbox>
          </Form.Item>
          <Form.Item shouldUpdate>
            {() => {
              const inherit = regionChannelForm.getFieldValue('inherit')
              if (inherit) {
                return <Text type="secondary">将使用应用级当前生效模板</Text>
              }
              return (
                <Row gutter={[16, 16]}>
                  <Col xs={24} md={12}>
                    <Form.Item name="template_id" label="选择模板">
                      <Select
                        placeholder="请选择模板"
                        options={regionTemplates.map((t) => ({ label: t.name, value: t.id }))}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="template_name" label="自定义模板名称(可选)">
                      <Input placeholder="未选择模板时生效" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card size="small" title="白名单" style={{ borderRadius: 8 }}>
                      <Form.Item name="allow_countries" label="国家 (ISO)">
                        <Select
                          mode="tags"
                          placeholder="例如：CN, US"
                          options={regionOptions.countries.map((c) => ({ label: c, value: c }))}
                          optionFilterProp="label"
                        />
                      </Form.Item>
                      <Form.Item shouldUpdate>
                        {() => {
                          const countries = regionChannelForm.getFieldValue('allow_countries') || []
                          const filtered = filterRegionOptions(countries, regionOptions.provinces)
                          const emptyMsg = countries.length === 0
                            ? '请先选择国家'
                            : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                          return (
                            <Form.Item name="allow_provinces" label="省/州 (ISO|省)">
                              <Select
                                mode="tags"
                                placeholder="例如：CN|广东, US|California"
                                options={filtered.map((p) => ({ label: formatRegionLabel(p), value: p }))}
                                optionFilterProp="label"
                                notFoundContent={emptyMsg || '无数据'}
                              />
                            </Form.Item>
                          )
                        }}
                      </Form.Item>
                      <Form.Item shouldUpdate>
                        {() => {
                          const countries = regionChannelForm.getFieldValue('allow_countries') || []
                          const provinces = regionChannelForm.getFieldValue('allow_provinces') || []
                          const filtered = filterCityOptions(countries, provinces, regionOptions.cities)
                          const emptyMsg = countries.length === 0
                            ? '请先选择国家'
                            : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                          return (
                            <Form.Item name="allow_cities" label="城市 (ISO|省|市)">
                              <Select
                                mode="tags"
                                placeholder="例如：CN|广东|深圳, US|California|San Francisco"
                                options={filtered.map((c) => ({ label: formatCityLabel(c), value: c }))}
                                optionFilterProp="label"
                                notFoundContent={emptyMsg || '无数据'}
                              />
                            </Form.Item>
                          )
                        }}
                      </Form.Item>
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card size="small" title="黑名单" style={{ borderRadius: 8 }}>
                      <Form.Item name="deny_countries" label="国家 (ISO)">
                        <Select
                          mode="tags"
                          placeholder="例如：RU"
                          options={regionOptions.countries.map((c) => ({ label: c, value: c }))}
                          optionFilterProp="label"
                        />
                      </Form.Item>
                      <Form.Item shouldUpdate>
                        {() => {
                          const countries = regionChannelForm.getFieldValue('deny_countries') || []
                          const filtered = filterRegionOptions(countries, regionOptions.provinces)
                          const emptyMsg = countries.length === 0
                            ? '请先选择国家'
                            : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                          return (
                            <Form.Item name="deny_provinces" label="省/州 (ISO|省)">
                              <Select
                                mode="tags"
                                placeholder="例如：CN|台湾"
                                options={filtered.map((p) => ({ label: formatRegionLabel(p), value: p }))}
                                optionFilterProp="label"
                                notFoundContent={emptyMsg || '无数据'}
                              />
                            </Form.Item>
                          )
                        }}
                      </Form.Item>
                      <Form.Item shouldUpdate>
                        {() => {
                          const countries = regionChannelForm.getFieldValue('deny_countries') || []
                          const provinces = regionChannelForm.getFieldValue('deny_provinces') || []
                          const filtered = filterCityOptions(countries, provinces, regionOptions.cities)
                          const emptyMsg = countries.length === 0
                            ? '请先选择国家'
                            : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                          return (
                            <Form.Item name="deny_cities" label="城市 (ISO|省|市)">
                              <Select
                                mode="tags"
                                placeholder="例如：CN|广东|深圳"
                                options={filtered.map((c) => ({ label: formatCityLabel(c), value: c }))}
                                optionFilterProp="label"
                                notFoundContent={emptyMsg || '无数据'}
                              />
                            </Form.Item>
                          )
                        }}
                      </Form.Item>
                    </Card>
                  </Col>
                </Row>
              )
            }}
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingRegionTemplate ? '编辑模板' : '新建模板'}
        open={regionTemplateOpen}
        onOk={saveRegionTemplate}
        onCancel={() => { setRegionTemplateOpen(false); setEditingRegionTemplate(null); regionTemplateForm.resetFields() }}
        width={720}
      >
        <Form layout="vertical" form={regionTemplateForm} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
            <Input placeholder="例如：国内白名单" />
          </Form.Item>
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}>
              <Card size="small" title="白名单" style={{ borderRadius: 8 }}>
                <Form.Item name="allow_countries" label="国家 (ISO)">
                  <Select
                    mode="tags"
                    placeholder="例如：CN, US"
                    options={regionOptions.countries.map((c) => ({ label: c, value: c }))}
                    optionFilterProp="label"
                  />
                </Form.Item>
                <Form.Item shouldUpdate>
                  {() => {
                    const countries = regionTemplateForm.getFieldValue('allow_countries') || []
                    const filtered = filterRegionOptions(countries, regionOptions.provinces)
                    const emptyMsg = countries.length === 0
                      ? '请先选择国家'
                      : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                    return (
                      <Form.Item name="allow_provinces" label="省/州 (ISO|省)">
                        <Select
                          mode="tags"
                          placeholder="例如：CN|广东, US|California"
                          options={filtered.map((p) => ({ label: formatRegionLabel(p), value: p }))}
                          optionFilterProp="label"
                          notFoundContent={emptyMsg || '无数据'}
                        />
                      </Form.Item>
                    )
                  }}
                </Form.Item>
                <Form.Item shouldUpdate>
                  {() => {
                    const countries = regionTemplateForm.getFieldValue('allow_countries') || []
                    const provinces = regionTemplateForm.getFieldValue('allow_provinces') || []
                    const filtered = filterCityOptions(countries, provinces, regionOptions.cities)
                    const emptyMsg = countries.length === 0
                      ? '请先选择国家'
                      : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                    return (
                      <Form.Item name="allow_cities" label="城市 (ISO|省|市)">
                        <Select
                          mode="tags"
                          placeholder="例如：CN|广东|深圳, US|California|San Francisco"
                          options={filtered.map((c) => ({ label: formatCityLabel(c), value: c }))}
                          optionFilterProp="label"
                          notFoundContent={emptyMsg || '无数据'}
                        />
                      </Form.Item>
                    )
                  }}
                </Form.Item>
              </Card>
            </Col>
            <Col xs={24} md={12}>
              <Card size="small" title="黑名单" style={{ borderRadius: 8 }}>
                <Form.Item name="deny_countries" label="国家 (ISO)">
                  <Select
                    mode="tags"
                    placeholder="例如：RU"
                    options={regionOptions.countries.map((c) => ({ label: c, value: c }))}
                    optionFilterProp="label"
                  />
                </Form.Item>
                <Form.Item shouldUpdate>
                  {() => {
                    const countries = regionTemplateForm.getFieldValue('deny_countries') || []
                    const filtered = filterRegionOptions(countries, regionOptions.provinces)
                    const emptyMsg = countries.length === 0
                      ? '请先选择国家'
                      : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                    return (
                      <Form.Item name="deny_provinces" label="省/州 (ISO|省)">
                        <Select
                          mode="tags"
                          placeholder="例如：CN|台湾"
                          options={filtered.map((p) => ({ label: formatRegionLabel(p), value: p }))}
                          optionFilterProp="label"
                          notFoundContent={emptyMsg || '无数据'}
                        />
                      </Form.Item>
                    )
                  }}
                </Form.Item>
                <Form.Item shouldUpdate>
                  {() => {
                    const countries = regionTemplateForm.getFieldValue('deny_countries') || []
                    const provinces = regionTemplateForm.getFieldValue('deny_provinces') || []
                    const filtered = filterCityOptions(countries, provinces, regionOptions.cities)
                    const emptyMsg = countries.length === 0
                      ? '请先选择国家'
                      : (filtered.length === 0 ? '当前区域暂不支持此区域划分' : undefined)
                    return (
                      <Form.Item name="deny_cities" label="城市 (ISO|省|市)">
                        <Select
                          mode="tags"
                          placeholder="例如：CN|广东|深圳"
                          options={filtered.map((c) => ({ label: formatCityLabel(c), value: c }))}
                          optionFilterProp="label"
                          notFoundContent={emptyMsg || '无数据'}
                        />
                      </Form.Item>
                    )
                  }}
                </Form.Item>
              </Card>
            </Col>
          </Row>
        </Form>
      </Modal>
    </Row>
  )
}
