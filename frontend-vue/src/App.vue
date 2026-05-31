<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  ArrowLeft,
  ArrowUp,
  Clock,
  Blocks,
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  CheckCircle2,
  CircleSlash,
  Crosshair,
  Database,
  Download,
  FileText,
  Hammer,
  Layers,
  LockKeyhole,
  LogOut,
  Play,
  RefreshCw,
  Search,
  Send,
  Eye,
  EyeOff,
  Save,
  Settings,
  Trash2,
  XCircle,
  UserRound
} from 'lucide-vue-next'
import { apiDelete, apiGet, apiPost, apiPostJson, apiPutJson, clearAuthToken, getAuthToken, setAuthToken } from './api'

const loading = ref(false)
const authChecked = ref(false)
const authenticated = ref(Boolean(getAuthToken()))
const loginLoading = ref(false)
const syncingFiles = ref(false)
const syncJob = ref(null)
const syncPageLoading = ref(false)
const remoteFiles = ref([])
const syncLocalFiles = ref([])
const syncSkipToday = ref(true)
const importSkipToday = ref(true)
const syncConfig = ref(null)
const settingsSaving = ref(false)
const settingsLoading = ref(false)
const smbTesting = ref(false)
const autoTaskLogs = ref([])
const autoTaskLogsLoading = ref(false)
const autoTaskLogsClearing = ref(false)
const showSmbPassword = ref(false)
const astrbotKey = ref(null)
const astrbotKeyLoading = ref(false)
const astrbotKeyResetting = ref(false)
const sourceFiles = ref({})
const sourceFilesLoading = ref({})
const importing = ref(false)
const error = ref('')
const loginError = ref('')
const importMessage = ref('')
const currentAdmin = ref('')
const currentAdminExpiresAt = ref(null)
const currentView = ref('dashboard')
const shareToken = ref('')
const shareData = ref(null)
const shareLoading = ref(false)
const shareError = ref('')
const xrayShareToken = ref('')
const xrayShareData = ref(null)
const xrayShareLoading = ref(false)
const xrayShareError = ref('')
const xrayShareEvidenceExpanded = ref(false)
const xrayShareEvidenceRowExpandedKeys = ref(new Set())
const xrayShareOrePositionsExpanded = ref(false)
const xrayShareOrePositionPage = ref(1)
const xrayShareOrePositionPageSize = ref(100)
const selectedShareServerId = ref('')
const rankingShareToken = ref('')
const rankingShareData = ref(null)
const rankingShareLoading = ref(false)
const rankingShareError = ref('')
const selectedRankingShareServerId = ref('')
const shareNow = ref(Date.now())
const lastOperation = ref(null)
const importFiles = ref([])
const importJob = ref(null)
const importPageLoading = ref(false)
const resetRemotePath = ref('')
const selectedResetKeys = ref(new Set())
const deletingLocalCsv = ref(false)
const selectedLocalDeleteKeys = ref(new Set())
const loginForm = ref({
  username: 'admin',
  password: ''
})
const passwordSaving = ref(false)
const showCurrentPassword = ref(false)
const showNewPassword = ref(false)
const showConfirmPassword = ref(false)
const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})
let importJobTimer = null
let syncJobTimer = null
let shareTimer = null
const serverTimeDisplay = ref('')
let serverTimeOffset = 0
let serverTimeTimer = null
const IMPORT_JOB_STORAGE_KEY = 'player-stats-last-import-jobs'
const SYNC_JOB_STORAGE_KEY = 'player-stats-last-sync-jobs'
const LOG_QUERY_FORM_STORAGE_KEY = 'player-stats-log-query-forms'
const LOG_TEXT_QUERY_FORM_STORAGE_KEY = 'player-stats-log-text-query-forms'
const LOG_XRAY_ANALYSIS_FORM_STORAGE_KEY = 'player-stats-log-xray-analysis-forms'
const LOG_QUERY_TYPE_COORDINATE = 'coordinate'
const LOG_QUERY_TYPE_PLAYER_KEYWORD = 'playerKeyword'
const LOG_QUERY_TYPE_XRAY = 'xrayAnalysis'
const LOG_QUERY_PAGE_SIZE_OPTIONS = [50, 100, 200, 500]
const XRAY_ORE_POSITION_PAGE_SIZE_OPTIONS = [50, 100, 200, 500]
const ACTIVE_JOB_STATUSES = ['PENDING', 'RUNNING']

const filters = ref({
  serverId: 'all',
  from: '',
  to: '',
  player: ''
})

const serverOptions = ref([
  { serverId: 'main', serverName: '主服' },
  { serverId: 'sub', serverName: '2服' }
])
const serverSummaries = ref([])
const overview = ref(emptyOverview())
const players = ref([])
const daily = ref([])
const imports = ref([])
const logQueryServerId = ref('main')
const logQueryMode = ref(LOG_QUERY_TYPE_COORDINATE)
const logQueryForms = ref(readSavedLogQueryForms())
const logTextQueryForms = ref(readSavedLogTextQueryForms())
const xrayAnalysisForms = ref(readSavedXrayAnalysisForms())
const logQueryResults = ref({})
const xrayAnalysisResults = ref({})
const logQueryLoading = ref({})
const xrayAnalysisLoading = ref({})
const logQueryPages = ref({})
const logQueryPageSizes = ref({})
let logQueryTimer = null

const selectedServerName = computed(() => {
  if (filters.value.serverId === 'all') return '合计'
  return serverOptions.value.find((item) => item.serverId === filters.value.serverId)?.serverName || filters.value.serverId
})

const serverTabs = computed(() => [
  { serverId: 'all', serverName: '合计' },
  ...serverOptions.value
])

const isSharePage = computed(() => Boolean(shareToken.value))
const isXraySharePage = computed(() => Boolean(xrayShareToken.value))
const isRankingSharePage = computed(() => Boolean(rankingShareToken.value))
const shareServers = computed(() => shareData.value?.servers || [])
const selectedShareServer = computed(() => {
  if (!shareServers.value.length) return null
  return shareServers.value.find((server) => server.serverId === selectedShareServerId.value) || shareServers.value[0]
})
const xraySharePlayer = computed(() => xrayShareData.value?.player || null)
const shareRemainingMs = computed(() => {
  if (!shareData.value?.expiresAt) return 0
  return Math.max(0, new Date(shareData.value.expiresAt).getTime() - shareNow.value)
})
const shareRemainingText = computed(() => {
  if (shareError.value) return '链接已过期'
  if (!shareData.value?.expiresAt) {
    return shareData.value?.ttlMinutes ? `链接${durationText(shareData.value.ttlMinutes)}内有效` : '链接有效期内可访问'
  }
  const totalSeconds = Math.ceil(shareRemainingMs.value / 1000)
  if (totalSeconds <= 0) return '链接已过期'
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes > 0) return `剩余 ${minutes} 分钟`
  return `剩余 ${seconds} 秒`
})
const xrayShareRemainingMs = computed(() => {
  if (!xrayShareData.value?.expiresAt) return 0
  return Math.max(0, new Date(xrayShareData.value.expiresAt).getTime() - shareNow.value)
})
const xrayShareRemainingText = computed(() => {
  if (xrayShareError.value) return '链接已过期'
  if (!xrayShareData.value?.expiresAt) {
    return xrayShareData.value?.ttlMinutes ? `链接 ${durationText(xrayShareData.value.ttlMinutes)}内有效` : '链接有效期内可访问'
  }
  const totalSeconds = Math.ceil(xrayShareRemainingMs.value / 1000)
  if (totalSeconds <= 0) return '链接已过期'
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes > 0) return `剩余 ${minutes} 分钟`
  return `剩余 ${seconds} 秒`
})
const rankingShareServers = computed(() => rankingShareData.value?.servers || [])
const selectedRankingShareServer = computed(() => {
  if (!rankingShareServers.value.length) return null
  return rankingShareServers.value.find((s) => s.serverId === selectedRankingShareServerId.value) || rankingShareServers.value[0]
})
const rankingShareRemainingMs = computed(() => {
  if (!rankingShareData.value?.expiresAt) return 0
  return Math.max(0, new Date(rankingShareData.value.expiresAt).getTime() - shareNow.value)
})
const rankingShareRemainingText = computed(() => {
  if (rankingShareError.value) return '链接已过期'
  if (!rankingShareData.value?.expiresAt) {
    return rankingShareData.value?.ttlMinutes ? `链接${durationText(rankingShareData.value.ttlMinutes)}内有效` : '链接有效期内可访问'
  }
  const ms = rankingShareRemainingMs.value
  if (ms <= 0) return '链接已过期'
  return `${durationText(Math.ceil(ms / 60000))}内有效`
})
const rankingShareTitle = computed(() => {
  if (rankingShareData.value?.rankingType === 'active') return '活跃榜'
  return '肝帝榜'
})

const maxDailyTotal = computed(() => {
  const max = Math.max(...daily.value.map((item) => item.totalCount || 0), 0)
  return max || 1
})

const leadingPlayers = computed(() => players.value.slice(0, 8))
const operationFiles = computed(() => lastOperation.value?.result?.files || [])
const importJobRunning = computed(() => importJob.value?.status === 'RUNNING' || importJob.value?.status === 'PENDING')
const importPageFiles = computed(() => importJobRunning.value ? (importJob.value?.files || importFiles.value) : importFiles.value)
const syncJobRunning = computed(() => syncJob.value?.status === 'RUNNING' || syncJob.value?.status === 'PENDING')
const selectedLogQueryResult = computed(() => logQueryResultFor(logQueryServerId.value) || null)
const selectedXrayAnalysisResult = computed(() => xrayAnalysisResultFor(logQueryServerId.value) || null)
const selectedLogFeatureResult = computed(() => (
  logQueryMode.value === LOG_QUERY_TYPE_XRAY ? selectedXrayAnalysisResult.value : selectedLogQueryResult.value
))
const selectedLogQueryRows = computed(() => selectedLogQueryResult.value?.rows || [])
const xrayDetailPlayerName = ref('')
const xrayDetailEvidenceExpanded = ref(false)
const xrayDetailEvidenceRowExpandedKeys = ref(new Set())
const xrayDetailOrePositionsExpanded = ref(false)
const xrayDetailOrePositionPage = ref(1)
const xrayDetailOrePositionPageSize = ref(100)
const xrayGroupSendDialogOpen = ref(false)
const xrayGroupSendLoading = ref(false)
const xrayGroupSendTtlText = ref('')
const xrayGroupSendMessage = ref('')
const xrayGroupSendError = ref('')
const selectedXrayDetailPlayer = computed(() => {
  if (!xrayDetailPlayerName.value) return null
  return (selectedXrayAnalysisResult.value?.players || [])
    .find((player) => player.playerName === xrayDetailPlayerName.value) || null
})
const xrayShareRareOreRows = computed(() => xraySharePlayer.value?.rareOreRows || [])
const xrayDetailRareOreRows = computed(() => selectedXrayDetailPlayer.value?.rareOreRows || [])
const xrayShareOrePositionTotalPages = computed(() => xrayPageCount(xrayShareRareOreRows.value.length, xrayShareOrePositionPageSize.value))
const xrayDetailOrePositionTotalPages = computed(() => xrayPageCount(xrayDetailRareOreRows.value.length, xrayDetailOrePositionPageSize.value))
const xraySharePagedRareOreRows = computed(() => pagedXrayRows(
  xrayShareRareOreRows.value,
  xrayShareOrePositionPage.value,
  xrayShareOrePositionPageSize.value
))
const xrayDetailPagedRareOreRows = computed(() => pagedXrayRows(
  xrayDetailRareOreRows.value,
  xrayDetailOrePositionPage.value,
  xrayDetailOrePositionPageSize.value
))
const xrayShareOrePositionOffset = computed(() => (clampPage(xrayShareOrePositionPage.value, xrayShareOrePositionTotalPages.value) - 1) * xrayShareOrePositionPageSize.value)
const xrayDetailOrePositionOffset = computed(() => (clampPage(xrayDetailOrePositionPage.value, xrayDetailOrePositionTotalPages.value) - 1) * xrayDetailOrePositionPageSize.value)
const xrayDetailResult = computed(() => xrayAnalysisResultFor(logQueryServerId.value) || null)
const selectedLogQueryRunning = computed(() => isActiveJob(selectedLogFeatureResult.value))
const selectedLogQueryLoading = computed(() => (
  logQueryMode.value === LOG_QUERY_TYPE_XRAY
    ? Boolean(xrayAnalysisLoading.value[logQueryStateKey(logQueryServerId.value, LOG_QUERY_TYPE_XRAY)])
    : Boolean(logQueryLoading.value[logQueryStateKey(logQueryServerId.value)])
))
const selectedLogQueryPage = computed(() => selectedLogQueryResult.value?.page || logQueryPages.value[logQueryStateKey(logQueryServerId.value)] || 1)
const selectedLogQueryPageSize = computed(() => selectedLogQueryResult.value?.pageSize || logQueryPageSizes.value[logQueryStateKey(logQueryServerId.value)] || 100)
const selectedLogQueryTotalPages = computed(() => selectedLogQueryResult.value?.totalPages || 0)
const syncPageFiles = computed(() => {
  const jobFiles = syncJob.value?.files || []
  if (!jobFiles.length) {
    return remoteFiles.value
  }

  const jobFileMap = new Map(jobFiles.map((file) => [fileCompareKey(file), file]))
  const mergedFiles = remoteFiles.value.map((remoteFile) => {
    const key = fileCompareKey(remoteFile)
    const jobFile = jobFileMap.get(key)
    if (!jobFile) {
      return remoteFile
    }
    jobFileMap.delete(key)
    return {
      ...remoteFile,
      ...jobFile,
      remotePath: remoteFile.remotePath || jobFile.remotePath,
      fileName: remoteFile.fileName || jobFile.fileName,
      fileSize: jobFile.fileSize || remoteFile.fileSize
    }
  })

  return [...mergedFiles, ...jobFileMap.values()]
})
const syncLocalFileMap = computed(() => {
  const fileMap = new Map()
  for (const file of syncLocalFiles.value) {
    fileMap.set(fileCompareKey(file), file)
  }
  return fileMap
})
const deletableLocalCopyFiles = computed(() => syncPageFiles.value.filter(canDeleteLocalCopy))
const selectedLocalDeleteCount = computed(() => selectedLocalDeleteKeys.value.size)
const allLocalCopiesSelected = computed(() => {
  const files = deletableLocalCopyFiles.value
  return files.length > 0 && files.every((file) => selectedLocalDeleteKeys.value.has(localCopyKey(file)))
})
const eligibleImportFiles = computed(() => importPageFiles.value.filter(canDeleteImportRecord))
const selectedResetCount = computed(() => selectedResetKeys.value.size)
const allEligibleSelected = computed(() => {
  const files = eligibleImportFiles.value
  return files.length > 0 && files.every((file) => selectedResetKeys.value.has(importFileKey(file)))
})
const syncNavLabel = computed(() => syncJobRunning.value
  ? `查看复制状态 ${selectedServerName.value}`
  : `前往复制 CSV ${selectedServerName.value}`)
const importNavLabel = computed(() => importJobRunning.value
  ? `查看解析状态 ${selectedServerName.value}`
  : `前往解析 ${selectedServerName.value}`)

onMounted(() => {
  if (openSharePageFromPath()) {
    return
  }
  bootstrapAuth()
})

onUnmounted(() => {
  stopShareTicker()
  stopServerTimeTicker()
  stopImportJobPolling()
  stopSyncJobPolling()
  stopLogQueryPolling()
})

function openSharePageFromPath() {
  const xrayMatch = window.location.pathname.match(/^\/xray-share\/([^/?#]+)/)
  if (xrayMatch) {
    xrayShareToken.value = decodeURIComponent(xrayMatch[1])
    authChecked.value = true
    loadXrayShareDetails()
    return true
  }

  const rankingMatch = window.location.pathname.match(/^\/share\/ranking\/([^/?#]+)/)
  if (rankingMatch) {
    rankingShareToken.value = decodeURIComponent(rankingMatch[1])
    authChecked.value = true
    loadRankingShareDetails()
    return true
  }

  const match = window.location.pathname.match(/^\/share\/([^/?#]+)/)
  if (!match) {
    return false
  }
  shareToken.value = decodeURIComponent(match[1])
  authChecked.value = true
  loadShareDetails()
  return true
}

async function loadXrayShareDetails() {
  xrayShareLoading.value = true
  xrayShareError.value = ''
  try {
    const data = await apiGet(`/api/share/xray/${encodeURIComponent(xrayShareToken.value)}`, {}, { auth: false })
    xrayShareData.value = data
    xrayShareEvidenceExpanded.value = false
    xrayShareEvidenceRowExpandedKeys.value = new Set()
    xrayShareOrePositionsExpanded.value = false
    xrayShareOrePositionPage.value = 1
    startShareTicker()
  } catch (err) {
    xrayShareData.value = null
    xrayShareError.value = err.message || '链接已过期，请重新生成'
    stopShareTicker()
  } finally {
    xrayShareLoading.value = false
  }
}

async function loadRankingShareDetails() {
  rankingShareLoading.value = true
  rankingShareError.value = ''
  try {
    const data = await apiGet(`/api/share/ranking/${encodeURIComponent(rankingShareToken.value)}`, {}, { auth: false })
    rankingShareData.value = data
    selectedRankingShareServerId.value = data.servers?.[0]?.serverId || ''
    startShareTicker()
  } catch (err) {
    rankingShareData.value = null
    rankingShareError.value = err.message || '链接已过期，请重新在群里查询'
    stopShareTicker()
  } finally {
    rankingShareLoading.value = false
  }
}

async function loadShareDetails() {
  shareLoading.value = true
  shareError.value = ''
  try {
    const data = await apiGet(`/api/share/${encodeURIComponent(shareToken.value)}`, {}, { auth: false })
    shareData.value = data
    selectedShareServerId.value = data.servers?.[0]?.serverId || ''
    startShareTicker()
  } catch (err) {
    shareData.value = null
    shareError.value = err.message || '链接已过期，请重新在群里查询'
    stopShareTicker()
  } finally {
    shareLoading.value = false
  }
}

function startShareTicker() {
  stopShareTicker()
  shareNow.value = Date.now()
  shareTimer = window.setInterval(() => {
    shareNow.value = Date.now()
    if (shareData.value?.expiresAt && shareRemainingMs.value <= 0) {
      shareError.value = '链接已过期，请重新在群里查询'
      stopShareTicker()
    }
    if (xrayShareData.value?.expiresAt && xrayShareRemainingMs.value <= 0) {
      xrayShareError.value = '链接已过期，请重新生成'
      stopShareTicker()
    }
    if (rankingShareData.value?.expiresAt && rankingShareRemainingMs.value <= 0) {
      rankingShareError.value = '链接已过期，请重新在群里查询'
      stopShareTicker()
    }
  }, 1000)
}

function stopShareTicker() {
  if (shareTimer) {
    window.clearInterval(shareTimer)
    shareTimer = null
  }
}

async function bootstrapAuth() {
  if (!getAuthToken()) {
    authChecked.value = true
    return
  }

  try {
    const admin = await apiGet('/api/auth/me')
    currentAdmin.value = admin.username
    currentAdminExpiresAt.value = admin.expiresAt
    authenticated.value = true
    await syncServerTime()
    await loadAll()
    await openInitialViewFromHash()
  } catch (err) {
    clearAuthToken()
    authenticated.value = false
  } finally {
    authChecked.value = true
  }
}

async function login() {
  loginLoading.value = true
  loginError.value = ''
  try {
    const result = await apiPostJson('/api/auth/login', loginForm.value, { auth: false })
    setAuthToken(result.token)
    currentAdmin.value = result.username
    currentAdminExpiresAt.value = result.expiresAt
    authenticated.value = true
    await syncServerTime()
    await loadAll()
    await openInitialViewFromHash()
  } catch (err) {
    loginError.value = err.message
  } finally {
    loginLoading.value = false
  }
}

async function logout() {
  stopImportJobPolling()
  stopSyncJobPolling()
  stopServerTimeTicker()
  await apiPost('/api/auth/logout').catch(() => {})
  clearAuthToken()
  authenticated.value = false
  currentAdmin.value = ''
  currentAdminExpiresAt.value = null
  importMessage.value = ''
  error.value = ''
  loginForm.value.password = ''
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const baseParams = {
      serverId: filters.value.serverId === 'all' ? '' : filters.value.serverId,
      from: filters.value.from,
      to: filters.value.to,
      player: filters.value.player
    }
    const [optionsData, serverData, overviewData, playerData, dailyData, importData] = await Promise.all([
      apiGet('/api/stats/server-options'),
      apiGet('/api/stats/servers', { from: filters.value.from, to: filters.value.to }),
      apiGet('/api/stats/overview', baseParams),
      apiGet('/api/stats/players', { ...baseParams, limit: 100 }),
      apiGet('/api/stats/daily', baseParams),
      apiGet('/api/stats/imports', { serverId: baseParams.serverId, limit: 20 })
    ])
    const options = asArray(optionsData)
    serverOptions.value = options.length ? options : serverOptions.value
    serverSummaries.value = asArray(serverData)
    overview.value = overviewData || emptyOverview()
    players.value = asArray(playerData)
    daily.value = asArray(dailyData)
    imports.value = asArray(importData)
  } catch (err) {
    handleApiError(err)
  } finally {
    loading.value = false
  }
}

async function runImport() {
  await startImportJob()
}

async function openImportPage() {
  if (filters.value.serverId === 'all') {
    error.value = '请先选择主服或 2服，再解析 CSV'
    currentView.value = 'dashboard'
    window.location.hash = ''
    return
  }

  currentView.value = 'importPage'
  window.location.hash = '#/import'
  stopImportJobPolling()
  importJob.value = null
  importing.value = false
  importMessage.value = ''
  error.value = ''
  await loadImportFiles()
  await restoreImportJob()
}

async function loadImportFiles({ resetJob = false } = {}) {
  importPageLoading.value = true
  error.value = ''
  try {
    importFiles.value = asArray(await apiGet('/api/import/files', {
      serverId: filters.value.serverId === 'all' ? '' : filters.value.serverId
    }))
    if (resetJob) {
      importJob.value = null
      importing.value = false
    }
    selectedResetKeys.value = new Set()
  } catch (err) {
    handleApiError(err)
  } finally {
    importPageLoading.value = false
  }
}

async function startImportJob() {
  if (filters.value.serverId === 'all') {
    error.value = '请先选择主服或 2服，再解析 CSV'
    return
  }

  importing.value = true
  error.value = ''
  importMessage.value = ''
  try {
    const job = await apiPost('/api/import/jobs', {
      serverId: filters.value.serverId === 'all' ? '' : filters.value.serverId,
      skipToday: importSkipToday.value
    })
    importJob.value = job
    saveJobId('import', job.jobId)
    importMessage.value = '解析任务已开始'
    pollImportJob(job.jobId)
  } catch (err) {
    handleApiError(err)
  } finally {
    if (!importJobRunning.value) {
      importing.value = false
    }
  }
}

function pollImportJob(jobId) {
  stopImportJobPolling()
  importJobTimer = window.setInterval(async () => {
    try {
      const job = await apiGet(`/api/import/jobs/${jobId}`)
      importJob.value = job
      if (!isActiveJob(job)) {
        stopImportJobPolling()
        importing.value = false
        importMessage.value = jobSummaryMessage('import', job)
        await loadAll()
        await loadImportFiles()
      }
    } catch (err) {
      stopImportJobPolling()
      importing.value = false
      handleApiError(err)
    }
  }, 1000)
}

function stopImportJobPolling() {
  if (importJobTimer) {
    window.clearInterval(importJobTimer)
    importJobTimer = null
  }
}

async function restoreImportJob() {
  const jobId = savedJobId('import')
  if (!jobId) {
    return
  }

  try {
    const job = await apiGet(`/api/import/jobs/${jobId}`)
    importJob.value = job
    importing.value = isActiveJob(job)
    importMessage.value = isActiveJob(job) ? '解析任务进行中' : jobSummaryMessage('import', job)
    if (isActiveJob(job)) {
      pollImportJob(job.jobId)
    }
  } catch (err) {
    clearSavedJobId('import')
  }
}

async function deleteImportRecord(file) {
  if (!file?.imported && !['IMPORTED', 'CHANGED', 'NEEDS_IMPORT'].includes(file?.status)) {
    return
  }
  const ok = window.confirm(`只删除数据库中的解析记录，不删除本地 CSV：\n${fileNameFromPath(file.remotePath)}\n\n删除后可以重新解析这个文件。确定继续吗？`)
  if (!ok) {
    return
  }

  resetRemotePath.value = file.remotePath
  error.value = ''
  try {
    await apiDelete('/api/import/files', {
      serverId: file.serverId,
      remotePath: file.remotePath
    })
    importMessage.value = '已删除数据库解析记录，可以重新解析'
    await loadImportFiles({ resetJob: true })
    await loadAll()
  } catch (err) {
    handleApiError(err)
  } finally {
    resetRemotePath.value = ''
  }
}

async function deleteSelectedImportRecords() {
  const files = importPageFiles.value.filter((file) => selectedResetKeys.value.has(importFileKey(file)))
  if (!files.length) {
    return
  }

  const ok = window.confirm(`只删除数据库中的解析记录，不删除本地 CSV。\n本次将删除 ${files.length} 个文件的解析记录，删除后可以重新解析。确定继续吗？`)
  if (!ok) {
    return
  }

  resetRemotePath.value = '__batch__'
  error.value = ''
  try {
    const results = await apiPostJson('/api/import/files/delete-records', {
      files: files.map((file) => ({
        serverId: file.serverId,
        remotePath: file.remotePath
      }))
    })
    const deletedCount = results.filter((result) => result.deleted).length
    const failedCount = results.length - deletedCount
    importMessage.value = `已删除 ${deletedCount} 个数据库解析记录${failedCount ? `，失败 ${failedCount} 个` : ''}`
    selectedResetKeys.value = new Set()
    await loadImportFiles({ resetJob: true })
    await loadAll()
  } catch (err) {
    handleApiError(err)
  } finally {
    resetRemotePath.value = ''
  }
}

async function openSyncPage() {
  if (filters.value.serverId === 'all') {
    error.value = '请先选择主服或 2服，再复制 CSV'
    currentView.value = 'dashboard'
    window.location.hash = ''
    return
  }

  currentView.value = 'syncPage'
  window.location.hash = '#/sync'
  stopSyncJobPolling()
  syncJob.value = null
  syncingFiles.value = false
  remoteFiles.value = []
  syncLocalFiles.value = []
  selectedLocalDeleteKeys.value = new Set()
  importMessage.value = ''
  error.value = ''
  await loadSyncPageFiles()
  await restoreSyncJob()
}

async function loadSyncPageFiles() {
  syncPageLoading.value = true
  error.value = ''
  try {
    const params = {
      serverId: filters.value.serverId === 'all' ? '' : filters.value.serverId
    }
    const [remoteData, localData] = await Promise.all([
      apiGet('/api/import/remote-files', params),
      apiGet('/api/import/files', params)
    ])
    remoteFiles.value = asArray(remoteData)
    syncLocalFiles.value = asArray(localData)
    selectedLocalDeleteKeys.value = new Set()
  } catch (err) {
    handleApiError(err)
  } finally {
    syncPageLoading.value = false
  }
}

async function startSyncJob() {
  if (filters.value.serverId === 'all') {
    error.value = '请先选择主服或 2服，再复制 CSV'
    return
  }

  syncingFiles.value = true
  error.value = ''
  importMessage.value = ''
  try {
    const job = await apiPost('/api/import/sync-jobs', {
      serverId: filters.value.serverId === 'all' ? '' : filters.value.serverId,
      skipToday: syncSkipToday.value
    })
    syncJob.value = job
    saveJobId('sync', job.jobId)
    importMessage.value = '复制任务已开始'
    pollSyncJob(job.jobId)
  } catch (err) {
    handleApiError(err)
  } finally {
    if (!syncJobRunning.value) {
      syncingFiles.value = false
    }
  }
}

function pollSyncJob(jobId) {
  stopSyncJobPolling()
  syncJobTimer = window.setInterval(async () => {
    try {
      const job = await apiGet(`/api/import/sync-jobs/${jobId}`)
      syncJob.value = job
      if (!isActiveJob(job)) {
        stopSyncJobPolling()
        syncingFiles.value = false
        importMessage.value = jobSummaryMessage('sync', job)
        await loadSyncLocalFiles()
      }
    } catch (err) {
      stopSyncJobPolling()
      syncingFiles.value = false
      handleApiError(err)
    }
  }, 1000)
}

function stopSyncJobPolling() {
  if (syncJobTimer) {
    window.clearInterval(syncJobTimer)
    syncJobTimer = null
  }
}

async function loadSyncLocalFiles() {
  try {
    syncLocalFiles.value = asArray(await apiGet('/api/import/files', {
      serverId: filters.value.serverId === 'all' ? '' : filters.value.serverId
    }))
    selectedLocalDeleteKeys.value = new Set()
  } catch (err) {
    handleApiError(err)
  }
}

async function deleteSelectedLocalCsv() {
  const files = syncPageFiles.value.filter((file) => selectedLocalDeleteKeys.value.has(localCopyKey(file)))
  if (!files.length) {
    return
  }

  const ok = window.confirm(`只删除后端本地已复制的 CSV，不删除远程 SMB 文件。\n本次将删除 ${files.length} 个本地 CSV。确定继续吗？`)
  if (!ok) {
    return
  }

  deletingLocalCsv.value = true
  error.value = ''
  importMessage.value = ''
  try {
    const results = await apiPostJson('/api/import/local-files/delete', {
      files: files.map((file) => {
        const localFile = localFileForRemote(file)
        return {
          serverId: localFile.serverId,
          remotePath: localFile.remotePath
        }
      })
    })
    const deletedCount = results.filter((result) => result.deleted).length
    const failedCount = results.length - deletedCount
    importMessage.value = `已删除 ${deletedCount} 个本地 CSV${failedCount ? `，失败 ${failedCount} 个` : ''}`
    selectedLocalDeleteKeys.value = new Set()
    await loadSyncPageFiles()
  } catch (err) {
    handleApiError(err)
  } finally {
    deletingLocalCsv.value = false
  }
}

async function restoreSyncJob() {
  const jobId = savedJobId('sync')
  if (!jobId) {
    return
  }

  try {
    const job = await apiGet(`/api/import/sync-jobs/${jobId}`)
    syncJob.value = job
    syncingFiles.value = isActiveJob(job)
    importMessage.value = isActiveJob(job) ? '复制任务进行中' : jobSummaryMessage('sync', job)
    if (isActiveJob(job)) {
      pollSyncJob(job.jobId)
    }
  } catch (err) {
    clearSavedJobId('sync')
  }
}

async function openSettingsPage() {
  currentView.value = 'settingsPage'
  window.location.hash = '#/settings'
  error.value = ''
  importMessage.value = ''
  await Promise.all([loadSyncConfig(), loadAstrbotKey(), loadAutoTaskLogs()])
}

function openProfilePage() {
  currentView.value = 'profilePage'
  window.location.hash = '#/profile'
  error.value = ''
  importMessage.value = ''
  resetPasswordForm()
}

async function openLogQueryPage(serverId = filters.value.serverId, queryType = logQueryMode.value) {
  currentView.value = 'logQueryPage'
  window.location.hash = '#/log-query'
  error.value = ''
  importMessage.value = ''
  logQueryServerId.value = serverId && serverId !== 'all' ? serverId : 'main'
  logQueryMode.value = queryType
  closeXrayPlayerDetail()
  await loadAllLogQueryStates()
}

async function loadAllLogQueryStates(queryType = logQueryMode.value) {
  await Promise.all(serverOptions.value.map((server) => loadLogFeatureState(server.serverId, true, queryType)))
  startLogQueryPollingIfNeeded()
}

async function loadLogFeatureState(serverId, silent = false, queryType = logQueryMode.value) {
  if (queryType === LOG_QUERY_TYPE_XRAY) {
    return loadXrayAnalysisState(serverId, silent)
  }
  return loadLogQueryState(serverId, silent, queryType)
}

async function loadLogQueryState(serverId, silent = false, queryType = logQueryMode.value) {
  const key = logQueryStateKey(serverId, queryType)
  if (!silent) {
    logQueryLoading.value = { ...logQueryLoading.value, [key]: true }
  }
  try {
    const result = await apiGet('/api/import/log-query-jobs/latest', {
      serverId,
      queryType,
      page: logQueryPages.value[key] || 1,
      pageSize: logQueryPageSizes.value[key] || 100
    })
    logQueryPages.value = { ...logQueryPages.value, [key]: result.page || 1 }
    logQueryPageSizes.value = { ...logQueryPageSizes.value, [key]: result.pageSize || logQueryPageSizes.value[key] || 100 }
    logQueryResults.value = { ...logQueryResults.value, [key]: result }
  } catch (err) {
    handleApiError(err)
  } finally {
    if (!silent) {
      logQueryLoading.value = { ...logQueryLoading.value, [key]: false }
    }
  }
}

async function loadXrayAnalysisState(serverId, silent = false) {
  const key = logQueryStateKey(serverId, LOG_QUERY_TYPE_XRAY)
  if (!silent) {
    xrayAnalysisLoading.value = { ...xrayAnalysisLoading.value, [key]: true }
  }
  try {
    const result = await apiGet('/api/import/xray-analysis-jobs/latest', { serverId })
    xrayAnalysisResults.value = { ...xrayAnalysisResults.value, [key]: result }
  } catch (err) {
    handleApiError(err)
  } finally {
    if (!silent) {
      xrayAnalysisLoading.value = { ...xrayAnalysisLoading.value, [key]: false }
    }
  }
}

async function startLogQuery(serverId) {
  const queryType = logQueryMode.value
  const key = logQueryStateKey(serverId, queryType)
  error.value = ''
  importMessage.value = ''
  logQueryPages.value = { ...logQueryPages.value, [key]: 1 }
  if (queryType === LOG_QUERY_TYPE_XRAY) {
    closeXrayPlayerDetail()
    xrayAnalysisLoading.value = { ...xrayAnalysisLoading.value, [key]: true }
  } else {
    logQueryLoading.value = { ...logQueryLoading.value, [key]: true }
  }
  try {
    if (queryType === LOG_QUERY_TYPE_XRAY) {
      const result = await apiPostJson('/api/import/xray-analysis-jobs', xrayAnalysisRequestBody(serverId))
      xrayAnalysisResults.value = { ...xrayAnalysisResults.value, [key]: result }
      importMessage.value = `${serverNameById(serverId)} 矿透分析已开始`
    } else {
      const body = logQueryRequestBody(serverId, queryType)
      const result = await apiPostJson('/api/import/log-query-jobs', body)
      logQueryResults.value = { ...logQueryResults.value, [key]: result }
      importMessage.value = `${serverNameById(serverId)} 日志查询已开始`
    }
    startLogQueryPollingIfNeeded()
  } catch (err) {
    handleApiError(err)
  } finally {
    if (queryType === LOG_QUERY_TYPE_XRAY) {
      xrayAnalysisLoading.value = { ...xrayAnalysisLoading.value, [key]: false }
    } else {
      logQueryLoading.value = { ...logQueryLoading.value, [key]: false }
    }
  }
}

async function clearLogQuery(serverId) {
  const queryType = logQueryMode.value
  const key = logQueryStateKey(serverId, queryType)
  const ok = window.confirm(`确定清空 ${serverNameById(serverId)} 的${logQueryModeLabel(queryType)}结果吗？`)
  if (!ok) return
  error.value = ''
  importMessage.value = ''
  try {
    if (queryType === LOG_QUERY_TYPE_XRAY) {
      closeXrayPlayerDetail()
      const result = await apiDelete('/api/import/xray-analysis-jobs/latest', { serverId })
      xrayAnalysisResults.value = { ...xrayAnalysisResults.value, [key]: result }
    } else {
      const result = await apiDelete('/api/import/log-query-jobs/latest', { serverId, queryType })
      logQueryPages.value = { ...logQueryPages.value, [key]: 1 }
      logQueryResults.value = { ...logQueryResults.value, [key]: result }
    }
    importMessage.value = `${serverNameById(serverId)} 查询结果已清空`
  } catch (err) {
    handleApiError(err)
  }
}

function startLogQueryPollingIfNeeded() {
  if (!Object.values(logQueryResults.value).some(isActiveJob)
      && !Object.values(xrayAnalysisResults.value).some(isActiveJob)) {
    stopLogQueryPolling()
    return
  }
  if (logQueryTimer) {
    return
  }
  logQueryTimer = window.setInterval(async () => {
    const activeKeys = Object.entries(logQueryResults.value)
      .filter(([, result]) => isActiveJob(result))
      .map(([key]) => key)
    if (!activeKeys.length) {
      const activeXrayKeys = Object.entries(xrayAnalysisResults.value)
        .filter(([, result]) => isActiveJob(result))
        .map(([key]) => key)
      if (!activeXrayKeys.length) {
        stopLogQueryPolling()
        return
      }
      await Promise.all(activeXrayKeys.map((key) => {
        const [, serverId] = key.split(':')
        return loadXrayAnalysisState(serverId, true)
      }))
      return
    }
    const activeXrayKeys = Object.entries(xrayAnalysisResults.value)
      .filter(([, result]) => isActiveJob(result))
      .map(([key]) => key)
    await Promise.all(activeKeys.map((key) => {
      const [queryType, serverId] = key.split(':')
      return loadLogQueryState(serverId, true, queryType)
    }).concat(activeXrayKeys.map((key) => {
      const [, serverId] = key.split(':')
      return loadXrayAnalysisState(serverId, true)
    })))
  }, 1800)
}

function stopLogQueryPolling() {
  if (logQueryTimer) {
    window.clearInterval(logQueryTimer)
    logQueryTimer = null
  }
}

function setLogQueryServer(serverId) {
  closeXrayPlayerDetail()
  logQueryServerId.value = serverId
  loadLogFeatureState(serverId, true)
}

function setLogQueryMode(queryType) {
  if (logQueryMode.value === queryType) return
  logQueryMode.value = queryType
  closeXrayPlayerDetail()
  error.value = ''
  importMessage.value = ''
  loadAllLogQueryStates(queryType)
}

function openXrayPlayerDetail(player) {
  if (!player?.playerName) return
  xrayDetailPlayerName.value = player.playerName
  xrayDetailEvidenceExpanded.value = false
  xrayDetailEvidenceRowExpandedKeys.value = new Set()
  xrayDetailOrePositionsExpanded.value = false
  xrayDetailOrePositionPage.value = 1
  closeXrayGroupSendDialog()
  currentView.value = 'xrayDetailPage'
  logQueryMode.value = LOG_QUERY_TYPE_XRAY
  window.location.hash = `#/log-query/xray-detail?serverId=${encodeURIComponent(logQueryServerId.value)}&player=${encodeURIComponent(player.playerName)}`
}

function closeXrayPlayerDetail() {
  xrayDetailPlayerName.value = ''
  xrayDetailEvidenceExpanded.value = false
  xrayDetailEvidenceRowExpandedKeys.value = new Set()
  xrayDetailOrePositionsExpanded.value = false
  xrayDetailOrePositionPage.value = 1
  closeXrayGroupSendDialog()
}

async function openXrayDetailPage(serverId, playerName) {
  logQueryServerId.value = serverId && serverId !== 'all' ? serverId : 'main'
  logQueryMode.value = LOG_QUERY_TYPE_XRAY
  xrayDetailPlayerName.value = playerName || ''
  xrayDetailEvidenceExpanded.value = false
  xrayDetailEvidenceRowExpandedKeys.value = new Set()
  xrayDetailOrePositionsExpanded.value = false
  xrayDetailOrePositionPage.value = 1
  closeXrayGroupSendDialog()
  currentView.value = 'xrayDetailPage'
  error.value = ''
  importMessage.value = ''
  await loadXrayAnalysisState(logQueryServerId.value)
  startLogQueryPollingIfNeeded()
}

function openXrayGroupSendDialog() {
  xrayGroupSendTtlText.value = ''
  xrayGroupSendMessage.value = ''
  xrayGroupSendError.value = ''
  xrayGroupSendDialogOpen.value = true
}

function closeXrayGroupSendDialog() {
  xrayGroupSendDialogOpen.value = false
  xrayGroupSendLoading.value = false
  xrayGroupSendTtlText.value = ''
  xrayGroupSendMessage.value = ''
  xrayGroupSendError.value = ''
}

async function sendXrayDetailToGroup() {
  if (!selectedXrayDetailPlayer.value?.playerName) return
  xrayGroupSendLoading.value = true
  xrayGroupSendMessage.value = ''
  xrayGroupSendError.value = ''
  try {
    const ttlMinutes = parseShareDurationMinutes(xrayGroupSendTtlText.value)
    const result = await apiPostJson('/api/share/xray/send-to-group', {
      serverId: logQueryServerId.value,
      serverName: serverNameById(logQueryServerId.value),
      fromTime: xrayDetailResult.value?.fromTime || '',
      toTime: xrayDetailResult.value?.toTime || '',
      playerName: selectedXrayDetailPlayer.value.playerName,
      ttlMinutes,
      player: selectedXrayDetailPlayer.value
    })
    xrayGroupSendMessage.value = `已提交发送任务，链接 ${durationText(result.ttlMinutes || ttlMinutes)}内有效`
    importMessage.value = '已提交给 AstrBot 插件，请确认插件已启用群发送并填好群号'
  } catch (err) {
    xrayGroupSendError.value = err.message || '发送任务提交失败'
  } finally {
    xrayGroupSendLoading.value = false
  }
}

async function backToXrayAnalysis() {
  await openLogQueryPage(logQueryServerId.value, LOG_QUERY_TYPE_XRAY)
}

function logQueryStateKey(serverId, queryType = logQueryMode.value) {
  return `${queryType}:${serverId}`
}

function logQueryResultFor(serverId, queryType = logQueryMode.value) {
  return logQueryResults.value[logQueryStateKey(serverId, queryType)]
}

function xrayAnalysisResultFor(serverId) {
  return xrayAnalysisResults.value[logQueryStateKey(serverId, LOG_QUERY_TYPE_XRAY)]
}

function logFeatureResultFor(serverId, queryType = logQueryMode.value) {
  return queryType === LOG_QUERY_TYPE_XRAY ? xrayAnalysisResultFor(serverId) : logQueryResultFor(serverId, queryType)
}

function logQueryModeLabel(queryType = logQueryMode.value) {
  if (queryType === LOG_QUERY_TYPE_PLAYER_KEYWORD) return '综合筛选'
  if (queryType === LOG_QUERY_TYPE_XRAY) return '矿透分析'
  return '通过坐标查询'
}

function setLogQueryPage(page) {
  const key = logQueryStateKey(logQueryServerId.value)
  const totalPages = selectedLogQueryTotalPages.value || 1
  const nextPage = Math.max(1, Math.min(Number(page) || 1, totalPages))
  if (nextPage === selectedLogQueryPage.value) return
  logQueryPages.value = { ...logQueryPages.value, [key]: nextPage }
  loadLogQueryState(logQueryServerId.value)
}

function setLogQueryPageSize(pageSize) {
  const key = logQueryStateKey(logQueryServerId.value)
  const safePageSize = LOG_QUERY_PAGE_SIZE_OPTIONS.includes(Number(pageSize)) ? Number(pageSize) : 100
  logQueryPageSizes.value = { ...logQueryPageSizes.value, [key]: safePageSize }
  logQueryPages.value = { ...logQueryPages.value, [key]: 1 }
  loadLogQueryState(logQueryServerId.value)
}

function scrollLogQueryToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function xrayAnalysisRequestBody(serverId) {
  const form = xrayAnalysisForms.value[serverId] || defaultXrayAnalysisForm()
  const fromDate = form.fromDate || (form.fromTime || '').slice(0, 10)
  const toDate = form.toDate || (form.toTime || '').slice(0, 10)
  saveXrayAnalysisForms()
  return {
    serverId,
    fromTime: fromDate ? `${fromDate}T00:00:00` : '',
    toTime: toDate ? `${toDate}T23:59:59` : '',
    playerName: (form.playerName || '').trim(),
    dimension: (form.dimension || '').trim()
  }
}

function logQueryRequestBody(serverId, queryType) {
  if (queryType === LOG_QUERY_TYPE_PLAYER_KEYWORD) {
    const form = logTextQueryForms.value[serverId] || defaultLogTextQueryForm()
    const playerName = (form.playerName || '').trim()
    const keyword = (form.keyword || '').trim()
    const action = (form.action || '').trim()
    saveLogTextQueryForms()
    return {
      serverId,
      queryType,
      fromDate: form.fromDate,
      toDate: form.toDate,
      playerName,
      keyword,
      action
    }
  }

  const form = logQueryForms.value[serverId] || defaultLogQueryForm()
  saveLogQueryForms()
  return {
    serverId,
    queryType,
    fromDate: form.fromDate,
    toDate: form.toDate,
    x1: numberOrNull(form.x1),
    y1: numberOrNull(form.y1),
    z1: numberOrNull(form.z1),
    x2: numberOrNull(form.x2),
    y2: numberOrNull(form.y2),
    z2: numberOrNull(form.z2),
    dimension: form.dimension
  }
}

function readSavedLogQueryForms() {
  const defaults = Object.fromEntries(serverOptions.value.map((server) => [server.serverId, defaultLogQueryForm()]))
  try {
    const saved = JSON.parse(localStorage.getItem(LOG_QUERY_FORM_STORAGE_KEY) || '{}')
    for (const server of serverOptions.value) {
      defaults[server.serverId] = {
        ...defaults[server.serverId],
        ...(saved[server.serverId] || {}),
        fromDate: saved[server.serverId]?.fromDate || saved[server.serverId]?.date || defaults[server.serverId].fromDate,
        toDate: saved[server.serverId]?.toDate || saved[server.serverId]?.date || defaults[server.serverId].toDate
      }
    }
  } catch (err) {
    // 表单缓存坏了就回到默认值。
  }
  return defaults
}

function defaultLogQueryForm() {
  const today = todayDateInput()
  return {
    fromDate: today,
    toDate: today,
    x1: '',
    y1: '',
    z1: '',
    x2: '',
    y2: '',
    z2: '',
    dimension: ''
  }
}

function readSavedLogTextQueryForms() {
  const defaults = Object.fromEntries(serverOptions.value.map((server) => [server.serverId, defaultLogTextQueryForm()]))
  try {
    const saved = JSON.parse(localStorage.getItem(LOG_TEXT_QUERY_FORM_STORAGE_KEY) || '{}')
    for (const server of serverOptions.value) {
      defaults[server.serverId] = {
        ...defaults[server.serverId],
        ...(saved[server.serverId] || {})
      }
    }
  } catch (err) {
    // 表单缓存坏了就回到默认值。
  }
  return defaults
}

function defaultLogTextQueryForm() {
  return {
    fromDate: '',
    toDate: '',
    playerName: '',
    keyword: '',
    action: ''
  }
}

function readSavedXrayAnalysisForms() {
  const defaults = Object.fromEntries(serverOptions.value.map((server) => [server.serverId, defaultXrayAnalysisForm()]))
  try {
    const saved = JSON.parse(localStorage.getItem(LOG_XRAY_ANALYSIS_FORM_STORAGE_KEY) || '{}')
    for (const server of serverOptions.value) {
      const savedForm = saved[server.serverId] || {}
      defaults[server.serverId] = {
        ...defaults[server.serverId],
        ...savedForm,
        fromDate: savedForm.fromDate || (savedForm.fromTime || '').slice(0, 10) || defaults[server.serverId].fromDate,
        toDate: savedForm.toDate || (savedForm.toTime || '').slice(0, 10) || defaults[server.serverId].toDate
      }
    }
  } catch (err) {
    // 本地缓存坏了就回到默认值。
  }
  return defaults
}

function defaultXrayAnalysisForm() {
  const today = todayDateInput()
  return {
    fromDate: today,
    toDate: today,
    playerName: '',
    dimension: ''
  }
}

function saveLogQueryForms() {
  try {
    localStorage.setItem(LOG_QUERY_FORM_STORAGE_KEY, JSON.stringify(logQueryForms.value))
  } catch (err) {
    // 本地缓存失败不影响查询。
  }
}

function saveLogTextQueryForms() {
  try {
    localStorage.setItem(LOG_TEXT_QUERY_FORM_STORAGE_KEY, JSON.stringify(logTextQueryForms.value))
  } catch (err) {
    // 本地缓存失败不影响查询。
  }
}

function saveXrayAnalysisForms() {
  try {
    localStorage.setItem(LOG_XRAY_ANALYSIS_FORM_STORAGE_KEY, JSON.stringify(xrayAnalysisForms.value))
  } catch (err) {
    // 本地缓存失败不影响查询。
  }
}

function todayDateInput() {
  const date = new Date()
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function numberOrNull(value) {
  if (value === null || value === undefined || value === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function durationText(minutes) {
  const safeMinutes = Math.max(1, Math.round(Number(minutes) || 0))
  if (safeMinutes % 1440 === 0) return `${safeMinutes / 1440}天`
  if (safeMinutes % 60 === 0) return `${safeMinutes / 60}小时`
  if (safeMinutes > 60) {
    const hours = Math.floor(safeMinutes / 60)
    const restMinutes = safeMinutes % 60
    return `${hours}小时${restMinutes}分钟`
  }
  return `${safeMinutes}分钟`
}

function parseShareDurationMinutes(value) {
  const text = String(value || '').trim()
  if (!text) return 1440
  const match = text.match(/^(\d+(?:\.\d+)?)\s*(分钟|分|m|min|小时|时|h|天|日|d)?$/i)
  if (!match) {
    throw new Error('有效时间格式不对，可填 1天、6小时、30分钟；留空默认 1天')
  }
  const amount = Number(match[1])
  if (!Number.isFinite(amount) || amount <= 0) {
    throw new Error('有效时间必须大于 0')
  }
  const unit = (match[2] || '天').toLowerCase()
  let minutes = amount * 1440
  if (['分钟', '分', 'm', 'min'].includes(unit)) {
    minutes = amount
  } else if (['小时', '时', 'h'].includes(unit)) {
    minutes = amount * 60
  }
  return Math.max(5, Math.min(Math.round(minutes), 10080))
}

function serverNameById(serverId) {
  return serverOptions.value.find((server) => server.serverId === serverId)?.serverName || serverId
}

function resetPasswordForm() {
  passwordForm.value = {
    currentPassword: '',
    newPassword: '',
    confirmPassword: ''
  }
  showCurrentPassword.value = false
  showNewPassword.value = false
  showConfirmPassword.value = false
}

async function changePassword() {
  error.value = ''
  importMessage.value = ''

  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    error.value = '两次输入的新密码不一致'
    return
  }
  if (passwordForm.value.newPassword.length < 8) {
    error.value = '新密码至少 8 位'
    return
  }

  passwordSaving.value = true
  try {
    await apiPostJson('/api/auth/password', {
      currentPassword: passwordForm.value.currentPassword,
      newPassword: passwordForm.value.newPassword
    })
    importMessage.value = '密码已修改'
    resetPasswordForm()
  } catch (err) {
    handleApiError(err)
  } finally {
    passwordSaving.value = false
  }
}

async function loadSyncConfig() {
  settingsLoading.value = true
  error.value = ''
  try {
    syncConfig.value = normalizeSyncConfig(await apiGet('/api/config/sync'))
    loadAllSourceFiles()
  } catch (err) {
    handleApiError(err)
  } finally {
    settingsLoading.value = false
  }
}

function normalizeSyncConfig(config) {
  const normalized = {
    ...config,
    skipToday: config.skipToday !== false,
    shareTtlMinutes: normalizeShareTtlMinutes(config.shareTtlMinutes),
    autoTasks: Array.isArray(config.autoTasks) && config.autoTasks.length
      ? config.autoTasks.map(normalizeAutoTask)
      : (config.sources || []).map((source) => normalizeAutoTask({
          serverId: source.sourceId,
          serverName: source.sourceName,
          syncEnabled: Boolean(config.autoRun),
          syncTime: defaultSyncTime(source.sourceId),
          importEnabled: Boolean(config.autoRun),
          importTime: cronToTime(config.syncCron) || defaultImportTime(source.sourceId)
        }))
  }
  return normalized
}

function normalizeAutoTask(task) {
  return {
    serverId: task.serverId || '',
    serverName: task.serverName || task.serverId || '',
    syncEnabled: Boolean(task.syncEnabled),
    syncTime: validTime(task.syncTime) || defaultSyncTime(task.serverId),
    importEnabled: Boolean(task.importEnabled),
    importTime: validTime(task.importTime) || defaultImportTime(task.serverId)
  }
}

function normalizeShareTtlMinutes(value) {
  const minutes = Number(value)
  if (!Number.isFinite(minutes) || minutes <= 0) return 60
  return Math.max(5, Math.min(Math.round(minutes), 10080))
}

function defaultSyncTime(serverId) {
  return serverId === 'sub' ? '00:40' : '00:20'
}

function defaultImportTime(serverId) {
  return serverId === 'sub' ? '00:50' : '00:30'
}

function validTime(value) {
  const text = String(value || '').trim()
  return /^\d{2}:\d{2}$/.test(text) ? text : ''
}

function cronToTime(cron) {
  const parts = String(cron || '').trim().split(/\s+/)
  if (parts.length < 3) return ''
  const minute = Number(parts[1])
  const hour = Number(parts[2])
  if (!Number.isInteger(minute) || !Number.isInteger(hour)) return ''
  if (minute < 0 || minute > 59 || hour < 0 || hour > 23) return ''
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

async function loadAstrbotKey() {
  astrbotKeyLoading.value = true
  error.value = ''
  try {
    astrbotKey.value = await apiGet('/api/config/astrbot-key')
  } catch (err) {
    handleApiError(err)
  } finally {
    astrbotKeyLoading.value = false
  }
}

async function loadAutoTaskLogs() {
  autoTaskLogsLoading.value = true
  try {
    autoTaskLogs.value = asArray(await apiGet('/api/import/auto-task-logs', { limit: 80 }))
  } catch (err) {
    handleApiError(err)
  } finally {
    autoTaskLogsLoading.value = false
  }
}

async function clearAutoTaskLogs() {
  const ok = window.confirm('确定清空所有自动任务日志吗？这个操作不会影响已复制的 CSV 或已入库数据。')
  if (!ok) {
    return
  }

  autoTaskLogsClearing.value = true
  error.value = ''
  importMessage.value = ''
  try {
    await apiDelete('/api/import/auto-task-logs')
    autoTaskLogs.value = []
    importMessage.value = '自动任务日志已清空'
  } catch (err) {
    handleApiError(err)
  } finally {
    autoTaskLogsClearing.value = false
  }
}

async function resetAstrbotKey() {
  const ok = window.confirm('重置后旧的 AstrBot 插件密钥会立即失效，需要同步更新 AstrBot 插件配置。确定继续吗？')
  if (!ok) {
    return
  }

  astrbotKeyResetting.value = true
  error.value = ''
  importMessage.value = ''
  try {
    astrbotKey.value = await apiPost('/api/config/astrbot-key/reset')
    importMessage.value = 'AstrBot 插件密钥已重置'
  } catch (err) {
    handleApiError(err)
  } finally {
    astrbotKeyResetting.value = false
  }
}

async function copyAstrbotKey() {
  if (!astrbotKey.value?.apiKey) {
    return
  }

  try {
    await navigator.clipboard.writeText(astrbotKey.value.apiKey)
    importMessage.value = 'AstrBot 插件密钥已复制'
  } catch (err) {
    error.value = '复制失败，请手动选中密钥复制'
  }
}

async function saveSyncConfig() {
  settingsSaving.value = true
  error.value = ''
  importMessage.value = ''
  try {
    syncConfig.value = normalizeSyncConfig(await apiPutJson('/api/config/sync', syncConfig.value))
    await loadAutoTaskLogs()
    importMessage.value = '配置已保存'
  } catch (err) {
    handleApiError(err)
  } finally {
    settingsSaving.value = false
  }
}

async function loadSourceFiles(sourceId) {
  sourceFilesLoading.value = { ...sourceFilesLoading.value, [sourceId]: true }
  try {
    const result = await apiGet('/api/config/source-files', { sourceId })
    if (result.success) {
      sourceFiles.value = { ...sourceFiles.value, [sourceId]: asArray(result.files) }
    } else {
      sourceFiles.value = { ...sourceFiles.value, [sourceId]: [] }
      error.value = result.message
    }
  } catch (err) {
    sourceFiles.value = { ...sourceFiles.value, [sourceId]: [] }
    handleApiError(err)
  } finally {
    sourceFilesLoading.value = { ...sourceFilesLoading.value, [sourceId]: false }
  }
}

async function loadAllSourceFiles() {
  if (!syncConfig.value?.sources) return
  for (const source of syncConfig.value.sources) {
    loadSourceFiles(source.sourceId)
  }
}

async function testSmbConnection() {
  smbTesting.value = true
  error.value = ''
  importMessage.value = ''
  try {
    const result = await apiPost('/api/config/smb-test')
    if (result.success) {
      importMessage.value = result.message
    } else {
      error.value = result.message
    }
  } catch (err) {
    handleApiError(err)
  } finally {
    smbTesting.value = false
  }
}

function showOperationResult(type, result) {
  lastOperation.value = {
    type,
    title: type === 'import' ? '解析入库结果' : 'CSV 复制结果',
    actionLabel: type === 'import' ? '导入' : '复制',
    result
  }
  currentView.value = 'operationResult'
  window.location.hash = type === 'import' ? '#/import-result' : '#/sync-result'
}

function goDashboard() {
  stopImportJobPolling()
  stopSyncJobPolling()
  stopLogQueryPolling()
  importing.value = false
  syncingFiles.value = false
  currentView.value = 'dashboard'
  window.location.hash = ''
}

function handleApiError(err) {
  error.value = err.message
  if (!getAuthToken()) {
    authenticated.value = false
    currentAdmin.value = ''
  }
}

function selectServer(serverId) {
  filters.value.serverId = serverId
  selectedResetKeys.value = new Set()
  importJob.value = null
  syncJob.value = null
  loadAll()
}

async function openInitialViewFromHash() {
  const hash = window.location.hash || ''
  if (hash.startsWith('#/log-query/xray-detail')) {
    const [, query = ''] = hash.split('?')
    const params = new URLSearchParams(query)
    await openXrayDetailPage(params.get('serverId') || filters.value.serverId, params.get('player') || '')
  } else if (hash === '#/sync') {
    await openSyncPage()
  } else if (hash === '#/import') {
    await openImportPage()
  } else if (hash === '#/profile') {
    openProfilePage()
  } else if (hash === '#/log-query') {
    await openLogQueryPage(filters.value.serverId)
  } else if (hash === '#/settings') {
    await openSettingsPage()
  }
}

function currentServerScope() {
  return filters.value.serverId === 'all' ? 'all' : filters.value.serverId
}

function jobStorageKey(type) {
  return type === 'sync' ? SYNC_JOB_STORAGE_KEY : IMPORT_JOB_STORAGE_KEY
}

function readSavedJobs(type) {
  try {
    return JSON.parse(localStorage.getItem(jobStorageKey(type)) || '{}')
  } catch (err) {
    return {}
  }
}

function savedJobId(type) {
  return readSavedJobs(type)[currentServerScope()] || ''
}

function saveJobId(type, jobId) {
  try {
    const jobs = readSavedJobs(type)
    jobs[currentServerScope()] = jobId
    localStorage.setItem(jobStorageKey(type), JSON.stringify(jobs))
  } catch (err) {
    // 保存状态失败不影响任务本身继续运行。
  }
}

function clearSavedJobId(type) {
  try {
    const jobs = readSavedJobs(type)
    delete jobs[currentServerScope()]
    localStorage.setItem(jobStorageKey(type), JSON.stringify(jobs))
  } catch (err) {
    // 忽略本地缓存清理失败。
  }
}

function isActiveJob(job) {
  return ACTIVE_JOB_STATUSES.includes(job?.status)
}

function logQueryStatusLabel(status) {
  const labels = {
    IDLE: '未查询',
    PENDING: '等待查询',
    RUNNING: '查询中',
    FINISHED: '查询完成',
    FINISHED_WITH_ERRORS: '部分文件失败',
    FAILED: '查询失败'
  }
  return labels[status] || status || '未查询'
}

function logQueryCoordinateText(x, y, z, dimension) {
  if (!x || x === '-') return '-'
  const dim = dimension && dimension !== '-' ? ` · ${dimension}` : ''
  return `${x}, ${y}, ${z}${dim}`
}

function rareOreDetailText(row) {
  const detail = [row?.detail1, row?.detail2]
    .find((value) => value && value !== '-')
  return detail || '-'
}

function xrayOreCounts(player) {
  return player?.ores?.length ? player.ores : (player?.rareOres || [])
}

function xrayAnalysisOreCounts(player) {
  return player?.analysisOres?.length ? player.analysisOres : xrayOreCounts(player)
}

function xrayAnalysisValue(player, analysisKey, fallbackKey = null) {
  const value = player?.[analysisKey]
  if (value !== undefined && value !== null) return value
  return fallbackKey ? (player?.[fallbackKey] || 0) : 0
}

function xrayAnalysisTrackingEvidenceCount(player) {
  return player?.analysisTrackingEvidenceCount ?? player?.evidence?.length ?? player?.trackingEvidenceCount ?? 0
}

function xrayPageCount(total, pageSize) {
  return Math.max(1, Math.ceil((Number(total) || 0) / (Number(pageSize) || 100)))
}

function clampPage(page, totalPages) {
  return Math.max(1, Math.min(Number(page) || 1, Number(totalPages) || 1))
}

function pagedXrayRows(rows, page, pageSize) {
  const totalPages = xrayPageCount(rows.length, pageSize)
  const safePage = clampPage(page, totalPages)
  const start = (safePage - 1) * pageSize
  return rows.slice(start, start + pageSize)
}

function setXrayShareOrePositionPage(page) {
  xrayShareOrePositionPage.value = clampPage(page, xrayShareOrePositionTotalPages.value)
}

function setXrayDetailOrePositionPage(page) {
  xrayDetailOrePositionPage.value = clampPage(page, xrayDetailOrePositionTotalPages.value)
}

function setXrayShareOrePositionPageSize(pageSize) {
  const safePageSize = XRAY_ORE_POSITION_PAGE_SIZE_OPTIONS.includes(Number(pageSize)) ? Number(pageSize) : 100
  xrayShareOrePositionPageSize.value = safePageSize
  xrayShareOrePositionPage.value = 1
}

function setXrayDetailOrePositionPageSize(pageSize) {
  const safePageSize = XRAY_ORE_POSITION_PAGE_SIZE_OPTIONS.includes(Number(pageSize)) ? Number(pageSize) : 100
  xrayDetailOrePositionPageSize.value = safePageSize
  xrayDetailOrePositionPage.value = 1
}

function toggleXrayShareOrePositions() {
  xrayShareOrePositionsExpanded.value = !xrayShareOrePositionsExpanded.value
  xrayShareOrePositionPage.value = 1
}

function toggleXrayDetailOrePositions() {
  xrayDetailOrePositionsExpanded.value = !xrayDetailOrePositionsExpanded.value
  xrayDetailOrePositionPage.value = 1
}

function xrayEvidenceRowKey(player, evidence, index) {
  return `${player?.playerName || ''}:${index}:${evidence?.startedAt || ''}:${evidence?.endedAt || ''}:${evidence?.summary || ''}`
}

function isShareXrayEvidenceRowsExpanded(evidence, index) {
  return xrayShareEvidenceRowExpandedKeys.value.has(xrayEvidenceRowKey(xraySharePlayer.value, evidence, index))
}

function toggleShareXrayEvidenceRows(evidence, index) {
  const key = xrayEvidenceRowKey(xraySharePlayer.value, evidence, index)
  const next = new Set(xrayShareEvidenceRowExpandedKeys.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  xrayShareEvidenceRowExpandedKeys.value = next
}

function isDetailXrayEvidenceRowsExpanded(evidence, index) {
  return xrayDetailEvidenceRowExpandedKeys.value.has(xrayEvidenceRowKey(selectedXrayDetailPlayer.value, evidence, index))
}

function toggleDetailXrayEvidenceRows(evidence, index) {
  const key = xrayEvidenceRowKey(selectedXrayDetailPlayer.value, evidence, index)
  const next = new Set(xrayDetailEvidenceRowExpandedKeys.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  xrayDetailEvidenceRowExpandedKeys.value = next
}

function formatDateOnly(value) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).format(new Date(value))
}

function xrayAnalysisDateRange(result) {
  if (!result?.fromTime || !result?.toTime) return '-'
  return `${formatDateOnly(result.fromTime)} - ${formatDateOnly(result.toTime)}`
}

function xrayRiskClass(score) {
  const value = Number(score) || 0
  if (value >= 85) return 'extreme'
  if (value >= 70) return 'high'
  if (value >= 45) return 'medium'
  if (value >= 20) return 'low'
  return 'watch'
}

function xrayEvidenceTime(evidence) {
  if (!evidence?.startedAt || !evidence?.endedAt) return '-'
  return `${formatDateTime(evidence.startedAt)} - ${formatDateTime(evidence.endedAt)}`
}

function xrayMiningSessionTime(player) {
  if (!player?.miningSessionStart || !player?.miningSessionEnd) return '-'
  return `${formatDateTime(player.miningSessionStart)} - ${formatDateTime(player.miningSessionEnd)}`
}

function xrayMiningSessionDuration(player) {
  if (!player?.miningSessionStart || !player?.miningSessionEnd) return '-'
  const minutes = Math.max(1, Math.round((new Date(player.miningSessionEnd).getTime() - new Date(player.miningSessionStart).getTime()) / 60000))
  return durationText(minutes)
}

function xrayRareRatioText(player) {
  const ratio = Number(player?.undergroundRareOreRatio)
  if (!Number.isFinite(ratio) || ratio <= 0) return '-'
  return `${(ratio * 100).toFixed(ratio >= 0.1 ? 1 : 2)}%`
}

function xrayAnalysisRareRatioText(player) {
  const ratio = Number(player?.analysisUndergroundRareOreRatio ?? player?.undergroundRareOreRatio)
  if (!Number.isFinite(ratio) || ratio <= 0) return '-'
  return `${(ratio * 100).toFixed(ratio >= 0.1 ? 1 : 2)}%`
}

function jobSummaryMessage(type, job) {
  if (job?.status === 'FAILED' && job.message) {
    return `${type === 'sync' ? '复制' : '解析'}任务失败：${job.message}`
  }
  const actionLabel = type === 'sync' ? '复制' : '导入'
  return `扫描 ${job?.scannedFiles || 0} 个文件，${actionLabel} ${job?.importedFiles || 0} 个，跳过 ${job?.skippedFiles || 0} 个，失败 ${job?.failedFiles || 0} 个`
}

async function syncServerTime() {
  try {
    const data = await apiGet('/api/stats/server-time')
    const serverMs = new Date(data.serverTime).getTime()
    serverTimeOffset = serverMs - Date.now()
    startServerTimeTicker()
  } catch (err) {
    serverTimeOffset = 0
    startServerTimeTicker()
  }
}

function startServerTimeTicker() {
  stopServerTimeTicker()
  tickServerTime()
  serverTimeTimer = window.setInterval(tickServerTime, 1000)
}

function stopServerTimeTicker() {
  if (serverTimeTimer) {
    window.clearInterval(serverTimeTimer)
    serverTimeTimer = null
  }
}

function tickServerTime() {
  const now = new Date(Date.now() + serverTimeOffset)
  serverTimeDisplay.value = now.getFullYear()
    + '-' + String(now.getMonth() + 1).padStart(2, '0')
    + '-' + String(now.getDate()).padStart(2, '0')
    + ' ' + String(now.getHours()).padStart(2, '0')
    + ':' + String(now.getMinutes()).padStart(2, '0')
    + ':' + String(now.getSeconds()).padStart(2, '0')
}

function emptyOverview() {
  return {
    playerCount: 0,
    brokenCount: 0,
    placedCount: 0,
    totalCount: 0,
    importedFileCount: 0,
    lastImportedAt: null
  }
}

function asArray(value) {
  return Array.isArray(value) ? value : []
}

function formatNumber(value) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

function formatDateTime(value) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

function formatShareMoment(value) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value)).replace('/', '月').replace(' ', '日 ')
}

function milestoneValue(item) {
  if (!item?.firstSeenAt) return item?.missingText || '还没有记录'
  const seenAt = formatShareMoment(item.firstSeenAt)
  return item.detail ? `${seenAt} ${item.detail}` : seenAt
}

function fileSize(value) {
  if (!value) return '0 B'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

function sourceStat(serverId, field) {
  return serverSummaries.value.find((item) => item.serverId === serverId)?.[field] || 0
}

function fileNameFromPath(path) {
  if (!path) return '-'
  const parts = path.split(/[\\/]/)
  return parts[parts.length - 1] || path
}

function statusLabel(status, context = 'import') {
  const labels = {
    READY: '就绪',
    PENDING: context === 'sync' ? '待复制' : '待解析',
    RUNNING: context === 'sync' ? '复制中' : '解析中',
    FINISHED: '已完成',
    FINISHED_WITH_ERRORS: '完成有失败',
    IMPORTED: '成功',
    COPIED: '已复制',
    SKIPPED: '跳过',
    SKIPPED_TODAY: '今日跳过',
    NEEDS_IMPORT: '需解析',
    CHANGED: '已变化',
    FAILED: '失败',
    REMOTE: '远程',
    INFO: '信息'
  }
  return labels[status] || status
}

function syncFileMessage(file) {
  if (file?.status === 'RUNNING' && (!file.message || file.message === '正在解析')) {
    return '正在复制'
  }
  return file?.message || '-'
}

function statusClass(status) {
  return {
    imported: status === 'IMPORTED' || status === 'COPIED' || status === 'FINISHED',
    running: status === 'RUNNING',
    pending: status === 'PENDING' || status === 'NEEDS_IMPORT' || status === 'CHANGED' || status === 'REMOTE',
    skipped: status === 'SKIPPED' || status === 'SKIPPED_TODAY' || status === 'INFO',
    failed: status === 'FAILED' || status === 'FINISHED_WITH_ERRORS'
  }
}

function fileCompareKey(file) {
  return `${file.serverId}\n${file.fileName || fileNameFromPath(file.remotePath)}`
}

function localFileForRemote(remoteFile) {
  return syncLocalFileMap.value.get(fileCompareKey(remoteFile))
}

function localCopyInfo(remoteFile) {
  const localFile = localFileForRemote(remoteFile)
  if (!localFile) {
    return {
      label: '未复制',
      className: 'pending',
      fileSize: 0,
      message: '本地未找到'
    }
  }

  const remoteSize = remoteFile?.fileSize || 0
  const localSize = localFile.fileSize || 0
  if (remoteSize && localSize && remoteSize !== localSize) {
    return {
      label: '大小不一致',
      className: 'failed',
      fileSize: localSize,
      message: `本地 ${fileSize(localSize)}，状态：${statusLabel(localFile.status)}`
    }
  }

  const remoteModified = new Date(remoteFile?.lastModified || 0).getTime()
  const localModified = new Date(localFile.lastModified || 0).getTime()
  if (remoteModified && localModified && remoteModified !== localModified) {
    return {
      label: '时间不一致',
      className: 'pending',
      fileSize: localSize,
      message: `本地 ${formatDateTime(localFile.lastModified)}，远程 ${formatDateTime(remoteFile.lastModified)}，状态：${statusLabel(localFile.status)}`
    }
  }

  return {
    label: '已在本地',
    className: 'imported',
    fileSize: localSize,
    message: `本地 ${fileSize(localSize)}，状态：${statusLabel(localFile.status)}`
  }
}

function canDeleteLocalCopy(file) {
  return Boolean(localFileForRemote(file)?.remotePath)
}

function localCopyKey(file) {
  const localFile = localFileForRemote(file)
  return localFile ? `${localFile.serverId}\n${localFile.remotePath}` : ''
}

function toggleLocalCopySelection(file, checked) {
  const key = localCopyKey(file)
  if (!key) {
    return
  }

  const next = new Set(selectedLocalDeleteKeys.value)
  if (checked) {
    next.add(key)
  } else {
    next.delete(key)
  }
  selectedLocalDeleteKeys.value = next
}

function toggleAllLocalCopySelection(checked) {
  selectedLocalDeleteKeys.value = checked
    ? new Set(deletableLocalCopyFiles.value.map(localCopyKey).filter(Boolean))
    : new Set()
}

function canDeleteImportRecord(file) {
  return Boolean(file?.imported || ['IMPORTED', 'CHANGED', 'NEEDS_IMPORT'].includes(file?.status))
}

function importFileKey(file) {
  return `${file.serverId}\n${file.remotePath}`
}

function toggleImportFileSelection(file, checked) {
  const next = new Set(selectedResetKeys.value)
  const key = importFileKey(file)
  if (checked) {
    next.add(key)
  } else {
    next.delete(key)
  }
  selectedResetKeys.value = next
}

function toggleAllImportFileSelection(checked) {
  selectedResetKeys.value = checked
    ? new Set(eligibleImportFiles.value.map(importFileKey))
    : new Set()
}
</script>

<template>
  <main v-if="isSharePage" class="share-shell">
    <section v-if="shareLoading" class="share-hero">
      <div class="share-hero-copy">
        <p class="eyebrow">PlayerLogger</p>
        <h1>正在打开足迹</h1>
        <p class="share-muted">正在读取玩家详情...</p>
      </div>
    </section>

    <section v-else-if="shareError" class="share-hero expired">
      <div class="share-hero-copy">
        <p class="eyebrow">PlayerLogger</p>
        <h1>链接已过期</h1>
        <p>{{ shareError }}</p>
      </div>
    </section>

    <template v-else-if="shareData && selectedShareServer">
      <section class="share-hero">
        <div class="share-hero-copy">
          <p class="eyebrow">玩家足迹</p>
          <h1>{{ shareData.playerName }}</h1>
          <p>{{ selectedShareServer.serverName }} · 总活动 {{ formatNumber(selectedShareServer.totalCount) }} 次</p>
        </div>
        <div class="share-hero-facts">
          <div>
            <span>当前服务器</span>
            <strong>{{ selectedShareServer.serverName }}</strong>
          </div>
          <div>
            <span>统计数据截止于</span>
            <strong>{{ shareData.latestLogDate || '-' }}</strong>
          </div>
          <div>
            <span>有效时间</span>
            <strong>{{ shareRemainingText }}</strong>
          </div>
        </div>
      </section>

      <section class="share-tabs" aria-label="服务器">
        <button
          v-for="server in shareServers"
          :key="server.serverId"
          type="button"
          :class="{ active: selectedShareServerId === server.serverId }"
          @click="selectedShareServerId = server.serverId"
        >
          {{ server.serverName }}
        </button>
      </section>

      <section class="share-dashboard">
        <aside class="share-side-column">
          <section class="share-panel share-summary-panel">
            <div class="share-section-title">
              <h2>总体统计</h2>
              <span>{{ selectedShareServer.serverName }}</span>
            </div>
            <div class="share-list">
              <div class="share-row">
                <span>破坏数量</span>
                <strong>{{ formatNumber(selectedShareServer.brokenCount) }}</strong>
              </div>
              <div class="share-row">
                <span>放置数量</span>
                <strong>{{ formatNumber(selectedShareServer.placedCount) }}</strong>
              </div>
              <div class="share-row">
                <span>总活动数量</span>
                <strong>{{ formatNumber(selectedShareServer.totalCount) }}</strong>
              </div>
              <div class="share-row">
                <span>首次记录时间</span>
                <strong>{{ formatDateTime(selectedShareServer.firstSeenAt) }}</strong>
              </div>
              <div class="share-row">
                <span>统计数据截止于</span>
                <strong>{{ selectedShareServer.latestLogDate || '-' }}</strong>
              </div>
            </div>
          </section>
        </aside>

        <div class="share-main-column">
          <section class="share-panel">
            <div class="share-section-title">
              <h2>生存足迹</h2>
              <span>{{ selectedShareServer.serverName }}</span>
            </div>
            <div class="share-milestone-grid">
              <div v-for="item in selectedShareServer.milestones" :key="item.type" class="share-milestone-card" :class="{ missing: !item.firstSeenAt }">
                <div>
                  <span>{{ item.label }}</span>
                  <strong>{{ milestoneValue(item) }}</strong>
                </div>
              </div>
            </div>
          </section>

          <div class="share-resource-layout">
            <section class="share-panel share-resource-panel share-resource-panel-wide">
              <div class="share-section-title">
                <h2>矿物收获</h2>
                <span>你一共挖到了</span>
              </div>
              <div class="share-ore-grid">
                <div v-for="ore in selectedShareServer.ores" :key="ore.type" class="share-ore-row" :class="{ 'is-zero': !ore.count }">
                  <span>{{ ore.label }}</span>
                  <strong>{{ ore.count ? formatNumber(ore.count) : '还没有记录' }}</strong>
                  <small v-if="ore.count">个</small>
                </div>
              </div>
            </section>

            <section class="share-panel share-resource-panel">
              <div class="share-section-title">
                <h2>砍树统计</h2>
                <span>你一共砍了</span>
              </div>
              <div class="share-ore-grid">
                <div v-for="wood in (selectedShareServer.woods || [])" :key="wood.type" class="share-ore-row" :class="{ 'is-zero': !wood.count }">
                  <span>{{ wood.label }}</span>
                  <strong>{{ wood.count ? formatNumber(wood.count) : '还没有记录' }}</strong>
                  <small v-if="wood.count">个</small>
                </div>
              </div>
            </section>

            <section class="share-panel share-resource-panel">
              <div class="share-section-title">
                <h2>栽树统计</h2>
                <span>你一共种了</span>
              </div>
              <div class="share-ore-grid">
                <div v-for="sapling in (selectedShareServer.saplings || [])" :key="sapling.type" class="share-ore-row" :class="{ 'is-zero': !sapling.count }">
                  <span>{{ sapling.label }}</span>
                  <strong>{{ sapling.count ? formatNumber(sapling.count) : '还没有记录' }}</strong>
                  <small v-if="sapling.count">棵</small>
                </div>
              </div>
            </section>
          </div>
        </div>
      </section>
    </template>
  </main>

  <main v-else-if="isXraySharePage" class="share-shell xray-share-shell">
    <section v-if="xrayShareLoading" class="share-hero">
      <div class="share-hero-copy">
        <p class="eyebrow">矿透分析</p>
        <h1>正在打开详情</h1>
        <p class="share-muted">正在读取矿透分析详情...</p>
      </div>
    </section>

    <section v-else-if="xrayShareError" class="share-hero expired">
      <div class="share-hero-copy">
        <p class="eyebrow">矿透分析</p>
        <h1>链接已过期</h1>
        <p>{{ xrayShareError }}</p>
      </div>
    </section>

    <template v-else-if="xrayShareData && xraySharePlayer">
      <section class="share-hero xray-share-hero">
        <div class="share-hero-copy">
          <p class="eyebrow">矿透分析详情</p>
          <h1>{{ xraySharePlayer.playerName }}</h1>
          <p>{{ xrayShareData.serverName }} · {{ xraySharePlayer.riskScore }} / 100 · {{ xraySharePlayer.riskLevel }}</p>
        </div>
        <div class="share-hero-facts">
          <div>
            <span>服务器</span>
            <strong>{{ xrayShareData.serverName }}</strong>
          </div>
          <div>
            <span>分析日期</span>
            <strong>{{ xrayAnalysisDateRange(xrayShareData) }}</strong>
          </div>
          <div>
            <span>有效时间</span>
            <strong>{{ xrayShareRemainingText }}</strong>
          </div>
        </div>
      </section>

      <article class="share-panel xray-share-detail">
        <div class="xray-detail-header">
          <div>
            <h3>{{ xraySharePlayer.playerName }}</h3>
            <span>{{ xraySharePlayer.riskScore }} / 100 · {{ xraySharePlayer.riskLevel }}</span>
          </div>
          <span class="risk-pill" :class="xrayRiskClass(xraySharePlayer.riskScore)">
            {{ xraySharePlayer.riskScore }} / 100
          </span>
        </div>

        <div class="xray-detail-body">
          <section class="xray-detail-section xray-toggle-section">
            <h4>最可疑挖矿会话</h4>
            <div class="xray-detail-grid">
              <div>
                <span>会话时间</span>
                <strong>{{ xrayMiningSessionTime(xraySharePlayer) }}</strong>
              </div>
              <div>
                <span>会话时长</span>
                <strong>{{ xrayMiningSessionDuration(xraySharePlayer) }}</strong>
              </div>
              <div>
                <span>会话地下破坏</span>
                <strong>{{ formatNumber(xraySharePlayer.miningSessionUndergroundBreaks || xraySharePlayer.miningSessionBreaks || 0) }}</strong>
              </div>
              <div>
                <span>地下稀有矿占比</span>
                <strong>{{ xrayRareRatioText(xraySharePlayer) }}</strong>
              </div>
              <div>
                <span>稀有矿脉</span>
                <strong>{{ formatNumber(xraySharePlayer.miningSessionRareVeins || 0) }}</strong>
              </div>
              <div>
                <span>十分钟矿脉峰值</span>
                <strong>{{ formatNumber(xraySharePlayer.peakRareVeinWindowCount || 0) }}</strong>
              </div>
            </div>
          </section>

          <section class="xray-detail-section xray-toggle-section">
            <h4>筛选周期总量</h4>
            <div class="xray-detail-grid">
              <div>
                <span>破坏方块</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisBreaks', 'miningSessionBreaks')) }}</strong>
              </div>
              <div>
                <span>地下破坏</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisUndergroundBreaks', 'miningSessionUndergroundBreaks')) }}</strong>
              </div>
              <div>
                <span>稀有矿</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisRareOreBreaks', 'miningSessionRareOreBreaks')) }}</strong>
              </div>
              <div>
                <span>钻石矿</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisDiamondOreBreaks', 'miningSessionDiamondOreBreaks')) }}</strong>
              </div>
              <div>
                <span>远古残骸</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisAncientDebrisBreaks', 'miningSessionAncientDebrisBreaks')) }}</strong>
              </div>
              <div>
                <span>地下稀有矿占比</span>
                <strong>{{ xrayAnalysisRareRatioText(xraySharePlayer) }}</strong>
              </div>
              <div>
                <span>稀有矿脉</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisRareVeins', 'miningSessionRareVeins')) }}</strong>
              </div>
              <div>
                <span>十分钟挖取峰值</span>
                <strong>{{ formatNumber(xrayAnalysisValue(xraySharePlayer, 'analysisPeakRareOreWindowCount', 'peakRareOreWindowCount')) }}</strong>
              </div>
              <div>
                <span>周期追矿证据</span>
                <strong>{{ formatNumber(xrayAnalysisTrackingEvidenceCount(xraySharePlayer)) }}</strong>
              </div>
            </div>
          </section>

          <section class="xray-detail-section xray-toggle-section">
            <h4>会话风险概况</h4>
            <div class="xray-detail-grid">
              <div>
                <span>稀有矿</span>
                <strong>{{ formatNumber(xraySharePlayer.miningSessionRareOreBreaks) }}</strong>
              </div>
              <div>
                <span>钻石矿</span>
                <strong>{{ formatNumber(xraySharePlayer.miningSessionDiamondOreBreaks) }}</strong>
              </div>
              <div>
                <span>远古残骸</span>
                <strong>{{ formatNumber(xraySharePlayer.miningSessionAncientDebrisBreaks) }}</strong>
              </div>
              <div>
                <span>会话追矿证据</span>
                <strong>{{ formatNumber(xraySharePlayer.trackingEvidenceCount) }}</strong>
              </div>
              <div>
                <span>十分钟挖取峰值</span>
                <strong>{{ formatNumber(xraySharePlayer.peakRareOreWindowCount || 0) }}</strong>
              </div>
              <div>
                <span>短时窗口</span>
                <strong v-if="xraySharePlayer.peakRareOreWindowStart && xraySharePlayer.peakRareOreWindowEnd">
                  {{ formatDateTime(xraySharePlayer.peakRareOreWindowStart) }} - {{ formatDateTime(xraySharePlayer.peakRareOreWindowEnd) }}
                </strong>
                <strong v-else>-</strong>
              </div>
            </div>
          </section>

          <section class="xray-detail-section xray-toggle-section">
            <h4>原因</h4>
            <div class="xray-chip-list">
              <span v-for="reason in xraySharePlayer.reasons" :key="reason">{{ reason }}</span>
              <span v-if="!xraySharePlayer.reasons?.length">-</span>
            </div>
          </section>

          <section class="xray-detail-section">
            <h4>筛选周期矿物明细</h4>
            <div class="xray-chip-list">
              <span v-for="ore in xrayAnalysisOreCounts(xraySharePlayer)" :key="ore.oreType">
                {{ ore.displayName }} x{{ formatNumber(ore.count) }}
              </span>
              <span v-if="!xrayAnalysisOreCounts(xraySharePlayer).length">-</span>
            </div>
          </section>

          <section class="xray-detail-section">
            <h4>筛选周期证据</h4>
            <div class="xray-expand-row">
              <button
                v-if="xraySharePlayer.evidence?.length"
                class="secondary-button mini-button"
                type="button"
                @click="xrayShareEvidenceExpanded = !xrayShareEvidenceExpanded"
              >
                <span>{{ xrayShareEvidenceExpanded ? '收起' : `展开 ${xraySharePlayer.evidence.length} 条` }}</span>
              </button>
            </div>
            <div v-if="xraySharePlayer.evidence?.length && xrayShareEvidenceExpanded" class="xray-evidence-list">
              <div v-for="(evidence, evidenceIndex) in xraySharePlayer.evidence" :key="`${xraySharePlayer.playerName}:${evidence.startedAt}:${evidence.summary}`" class="evidence-block">
                <strong>{{ evidence.summary }}</strong>
                <small>{{ xrayEvidenceTime(evidence) }} · +{{ evidence.score }}</small>
                <button
                  v-if="evidence.rows?.length"
                  class="secondary-button mini-button"
                  type="button"
                  @click="toggleShareXrayEvidenceRows(evidence, evidenceIndex)"
                >
                  <span>{{ isShareXrayEvidenceRowsExpanded(evidence, evidenceIndex) ? '收起明细' : `展开明细 ${evidence.rows.length} 条` }}</span>
                </button>
                <div v-if="evidence.rows?.length && isShareXrayEvidenceRowsExpanded(evidence, evidenceIndex)" class="evidence-lines">
                  <span v-for="row in evidence.rows" :key="`${row.filePath}:${row.lineNumber}`">
                    {{ row.time }} {{ row.detail1 || '-' }} @ {{ logQueryCoordinateText(row.x2, row.y2, row.z2, row.dimension2) }}
                  </span>
                </div>
              </div>
            </div>
            <p v-else-if="!xraySharePlayer.evidence?.length" class="settings-hint">没有追矿证据。</p>
          </section>

          <section class="xray-detail-section">
            <h4>稀有矿位置</h4>
            <div class="xray-expand-row">
              <button
                v-if="xraySharePlayer.rareOreRows?.length"
                class="secondary-button mini-button"
                type="button"
                @click="toggleXrayShareOrePositions"
              >
                <span>{{ xrayShareOrePositionsExpanded ? '收起' : `展开 ${xrayShareRareOreRows.length} 条` }}</span>
              </button>
            </div>
            <div v-if="xraySharePlayer.rareOreRows?.length && xrayShareOrePositionsExpanded" class="table-wrap">
              <div class="table-toolbar">
                <span>
                  第 {{ clampPage(xrayShareOrePositionPage, xrayShareOrePositionTotalPages) }} / {{ xrayShareOrePositionTotalPages }} 页，本页 {{ formatNumber(xraySharePagedRareOreRows.length) }} 条，共 {{ formatNumber(xrayShareRareOreRows.length) }} 条
                </span>
                <div class="pagination-controls">
                  <label>
                    <span>每页</span>
                    <select :value="xrayShareOrePositionPageSize" @change="setXrayShareOrePositionPageSize($event.target.value)">
                      <option v-for="size in XRAY_ORE_POSITION_PAGE_SIZE_OPTIONS" :key="size" :value="size">{{ size }}</option>
                    </select>
                  </label>
                  <button class="icon-button" type="button" title="上一页" :disabled="xrayShareOrePositionPage <= 1" @click="setXrayShareOrePositionPage(xrayShareOrePositionPage - 1)">
                    <ChevronLeft :size="18" />
                  </button>
                  <button class="icon-button" type="button" title="下一页" :disabled="xrayShareOrePositionPage >= xrayShareOrePositionTotalPages" @click="setXrayShareOrePositionPage(xrayShareOrePositionPage + 1)">
                    <ChevronRight :size="18" />
                  </button>
                </div>
              </div>
              <table class="xray-ore-position-table">
                <thead>
                  <tr>
                    <th>顺序</th>
                    <th>时间</th>
                    <th>矿物</th>
                    <th>玩家坐标</th>
                    <th>交互坐标</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in xraySharePagedRareOreRows" :key="`${row.filePath}:${row.lineNumber}:share-rare`">
                    <td>{{ xrayShareOrePositionOffset + index + 1 }}</td>
                    <td>{{ row.date }} {{ row.time }}</td>
                    <td class="message-cell">{{ rareOreDetailText(row) }}</td>
                    <td>{{ logQueryCoordinateText(row.x, row.y, row.z, row.dimension) }}</td>
                    <td>{{ logQueryCoordinateText(row.x2, row.y2, row.z2, row.dimension2) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-else-if="!xraySharePlayer.rareOreRows?.length" class="settings-hint">没有稀有矿位置记录。</p>
          </section>

          <div class="xray-detail-actions">
            <button class="secondary-button" type="button" @click="scrollLogQueryToTop">
              <ArrowUp :size="18" />
              <span>返回顶部</span>
            </button>
          </div>
        </div>
      </article>
    </template>
  </main>

  <main v-else-if="isRankingSharePage" class="share-shell">
    <section v-if="rankingShareLoading" class="share-hero">
      <div class="share-hero-copy">
        <p class="eyebrow">PlayerLogger</p>
        <h1>正在打开排行榜</h1>
        <p class="share-muted">正在读取排行数据...</p>
      </div>
    </section>

    <section v-else-if="rankingShareError" class="share-hero expired">
      <div class="share-hero-copy">
        <p class="eyebrow">PlayerLogger</p>
        <h1>链接已过期</h1>
        <p>{{ rankingShareError }}</p>
      </div>
    </section>

    <template v-else-if="rankingShareData && rankingShareServers.length">
      <section class="share-hero">
        <div class="share-hero-copy">
          <p class="eyebrow">{{ rankingShareTitle }}</p>
          <h1>{{ rankingShareTitle }} Top {{ rankingShareData.limit }}</h1>
          <p>按破坏+放置合计排序</p>
        </div>
        <div class="share-hero-facts">
          <div>
            <span>统计范围</span>
            <strong>{{ rankingShareData.fromDate || '最早' }} 至 {{ rankingShareData.toDate || '最新' }}</strong>
          </div>
          <div>
            <span>排名人数</span>
            <strong>Top {{ rankingShareData.limit }}</strong>
          </div>
          <div>
            <span>有效时间</span>
            <strong>{{ rankingShareRemainingText }}</strong>
          </div>
        </div>
      </section>

      <section class="share-tabs" aria-label="服务器">
        <button
          v-for="server in rankingShareServers"
          :key="server.serverId"
          type="button"
          :class="{ active: selectedRankingShareServerId === server.serverId }"
          @click="selectedRankingShareServerId = server.serverId"
        >
          {{ server.serverName }}
        </button>
      </section>

      <section class="share-dashboard">
        <article class="share-panel wide-panel">
          <div class="share-section-title">
            <h2>{{ selectedRankingShareServer?.serverName || '' }}</h2>
            <span>{{ selectedRankingShareServer?.players?.length || 0 }} 名玩家</span>
          </div>
          <div v-if="selectedRankingShareServer?.players?.length" class="ranking-card-grid">
            <div
              v-for="(player, index) in selectedRankingShareServer.players"
              :key="player.playerName"
              class="ranking-card"
            >
              <div class="ranking-card-head">
                <span class="ranking-card-rank">#{{ index + 1 }}</span>
                <span class="ranking-card-name">{{ player.playerName }}</span>
              </div>
              <div class="ranking-card-stats">
                <div class="ranking-card-stat">
                  <span>破坏</span>
                  <strong>{{ formatNumber(player.brokenCount) }}</strong>
                </div>
                <div class="ranking-card-stat">
                  <span>放置</span>
                  <strong>{{ formatNumber(player.placedCount) }}</strong>
                </div>
                <div class="ranking-card-stat">
                  <span>合计</span>
                  <strong>{{ formatNumber(player.totalCount) }}</strong>
                </div>
              </div>
            </div>
          </div>
          <p v-else class="empty">该服务器暂无数据</p>
        </article>
      </section>
    </template>
  </main>

  <main v-else-if="!authChecked" class="login-shell">
    <section class="login-panel">
      <LockKeyhole :size="28" />
      <p class="eyebrow">PlayerLogger</p>
      <h1>管理员后台</h1>
    </section>
  </main>

  <main v-else-if="!authenticated" class="login-shell">
    <form class="login-panel" @submit.prevent="login">
      <LockKeyhole :size="28" />
      <p class="eyebrow">PlayerLogger</p>
      <h1>管理员登录</h1>
      <label>
        <span>账号</span>
        <input v-model.trim="loginForm.username" type="text" autocomplete="username" />
      </label>
      <label>
        <span>密码</span>
        <input v-model="loginForm.password" type="password" autocomplete="current-password" />
      </label>
      <button class="primary-button" type="submit" :disabled="loginLoading">
        <LockKeyhole :size="18" />
        <span>{{ loginLoading ? '登录中' : '登录' }}</span>
      </button>
      <p v-if="loginError" class="notice error">{{ loginError }}</p>
    </form>
  </main>

  <main v-else class="app-shell">
    <section v-if="currentView === 'settingsPage'" class="result-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">PlayerLogger</p>
          <h1>系统设置</h1>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <button class="admin-badge" type="button" title="个人中心" @click="openProfilePage">
            <UserRound :size="16" />
            {{ currentAdmin }}
          </button>
          <button class="primary-button" type="button" :disabled="settingsSaving || settingsLoading" @click="saveSyncConfig">
            <Save :size="18" />
            <span>{{ settingsSaving ? '保存中' : '保存设置' }}</span>
          </button>
          <button class="secondary-button" type="button" @click="goDashboard">
            <ArrowLeft :size="18" />
            <span>返回统计</span>
          </button>
        </div>
      </div>

      <p v-if="error" class="notice error">{{ error }}</p>
      <p v-if="importMessage" class="notice success">{{ importMessage }}</p>

      <div v-if="syncConfig" class="settings-grid">
        <article class="panel wide">
          <div class="panel-title"><h2>SMB 服务器连接（只读）</h2></div>
          <div class="settings-form">
            <label>
              <span>主机地址</span>
              <input type="text" v-model="syncConfig.smbHost" placeholder="例: 192.168.1.100" />
            </label>
            <label>
              <span>端口</span>
              <input type="number" v-model.number="syncConfig.smbPort" placeholder="445" />
            </label>
            <label>
              <span>域</span>
              <input type="text" v-model="syncConfig.smbDomain" placeholder="可留空" />
            </label>
            <label>
              <span>用户名</span>
              <input type="text" v-model="syncConfig.smbUsername" placeholder="SMB 用户名" />
            </label>
            <label>
              <span>密码</span>
              <div class="password-field">
                <input :type="showSmbPassword ? 'text' : 'password'" v-model="syncConfig.smbPassword" placeholder="留空则不修改" />
                <button type="button" class="icon-button ghost password-toggle" @click="showSmbPassword = !showSmbPassword" tabindex="-1">
                  <EyeOff v-if="showSmbPassword" :size="16" />
                  <Eye v-else :size="16" />
                </button>
              </div>
            </label>
            <label>
              <span>共享名</span>
              <input type="text" v-model="syncConfig.smbShare" placeholder="例: share 或 data（Windows 共享文件夹名称）" />
            </label>
            <p class="settings-hint">共享名是 Windows 上共享的文件夹名称，右键文件夹 → 属性 → 共享 中可以看到。如果其它设备用 \\IP 访问，共享名就是 IP 后的部分。</p>
            <p class="settings-hint">连接为只读模式，不会修改或删除远程服务器上的文件。</p>
            <div class="settings-actions">
              <button class="secondary-button" type="button" :disabled="smbTesting" @click="testSmbConnection">
                <RefreshCw :size="16" :class="{ spin: smbTesting }" />
                <span>{{ smbTesting ? '测试中...' : '测试连接' }}</span>
              </button>
            </div>
          </div>
        </article>

        <article class="panel wide">
          <div class="panel-title">
            <h2>AstrBot 插件密钥</h2>
            <span>{{ astrbotKey?.headerName || 'X-Player-Stats-Key' }}</span>
          </div>
          <div class="settings-form">
            <label class="wide-field">
              <span>插件密钥</span>
              <input type="text" :value="astrbotKeyLoading ? '加载中...' : (astrbotKey?.apiKey || '')" readonly />
            </label>
            <label>
              <span>分享链接有效期（分钟）</span>
              <input type="number" min="5" max="10080" step="1" v-model.number="syncConfig.shareTtlMinutes" />
            </label>
            <p class="settings-hint">把这个密钥填到 AstrBot 插件配置的 api_key 中。插件只用它查询玩家统计，不需要管理员登录 token。</p>
            <p class="settings-hint">/我的游戏信息 生成的详细足迹链接会使用这个有效期，范围 5 分钟到 7 天。</p>
            <div class="settings-actions">
              <button class="secondary-button" type="button" :disabled="astrbotKeyLoading || !astrbotKey?.apiKey" @click="copyAstrbotKey">
                <FileText :size="16" />
                <span>复制密钥</span>
              </button>
              <button class="table-action danger" type="button" :disabled="astrbotKeyResetting" @click="resetAstrbotKey">
                <RefreshCw :size="16" :class="{ spin: astrbotKeyResetting }" />
                <span>{{ astrbotKeyResetting ? '重置中' : '重置密钥' }}</span>
              </button>
            </div>
          </div>
        </article>

        <article v-for="source in syncConfig.sources" :key="source.id" class="panel wide">
          <div class="panel-title">
            <h2>{{ source.sourceName || source.sourceId }} - SMB 目录</h2>
            <label class="toggle-label">
              <input type="checkbox" v-model="source.enabled" />
              <span>启用</span>
            </label>
          </div>
          <div class="settings-form">
            <label>
              <span>目录路径</span>
              <input type="text" v-model="source.smbDirectory" placeholder="例: logs/main" />
            </label>
            <label>
              <span>文件匹配</span>
              <input type="text" v-model="source.smbFileGlob" placeholder="例: player_actions_*.csv" />
            </label>
            <div class="settings-actions">
              <button class="secondary-button" type="button" :disabled="sourceFilesLoading[source.sourceId]" @click="loadSourceFiles(source.sourceId)">
                <RefreshCw :size="16" :class="{ spin: sourceFilesLoading[source.sourceId] }" />
                <span>{{ sourceFilesLoading[source.sourceId] ? '加载中...' : '刷新文件列表' }}</span>
              </button>
            </div>
          </div>
          <div v-if="sourceFiles[source.sourceId]?.length" class="source-file-list">
            <table class="data-table compact">
              <thead>
                <tr>
                  <th>文件名</th>
                  <th>大小</th>
                  <th>修改时间</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="file in sourceFiles[source.sourceId]" :key="file.remotePath">
                  <td class="path-cell">{{ file.fileName }}</td>
                  <td>{{ fileSize(file.fileSize) }}</td>
                  <td>{{ file.lastModified ? new Date(file.lastModified).toLocaleString('zh-CN') : '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else-if="sourceFiles[source.sourceId] && !sourceFilesLoading[source.sourceId]" class="settings-hint" style="padding: 0.75rem 1.25rem;">该目录下没有文件，请检查目录路径设置</p>
        </article>

        <article class="panel wide">
          <div class="panel-title"><h2>自动任务</h2></div>
          <div class="settings-form">
            <label class="toggle-label">
              <input type="checkbox" v-model="syncConfig.skipToday" />
              <span>默认跳过当天文件</span>
            </label>
            <p class="settings-hint">当天的日志文件仍在写入中，直接复制可能导致数据不完整。建议保持开启。</p>
          </div>
          <div class="auto-task-list">
            <div v-for="task in syncConfig.autoTasks" :key="task.serverId" class="auto-task-row">
              <div class="auto-task-server">
                <strong>{{ task.serverName || task.serverId }}</strong>
                <span>{{ task.serverId }}</span>
              </div>
              <label class="toggle-label">
                <input type="checkbox" v-model="task.syncEnabled" />
                <span>自动复制 CSV</span>
              </label>
              <label class="time-field">
                <span>复制时间</span>
                <input type="time" v-model="task.syncTime" :disabled="!task.syncEnabled" />
              </label>
              <label class="toggle-label">
                <input type="checkbox" v-model="task.importEnabled" />
                <span>自动解析入库</span>
              </label>
              <label class="time-field">
                <span>解析时间</span>
                <input type="time" v-model="task.importTime" :disabled="!task.importEnabled" />
              </label>
            </div>
          </div>
          <p class="settings-hint schedule-hint">建议先复制 CSV，再间隔 5-10 分钟解析入库；主服和 2服可以设置不同时间，避免任务同时抢占。</p>
        </article>

        <article class="panel wide">
          <div class="panel-title">
            <h2>自动任务日志</h2>
            <span>{{ autoTaskLogs.length }} 条</span>
          </div>
          <div class="bulk-actions">
            <button class="secondary-button" type="button" :disabled="autoTaskLogsLoading" @click="loadAutoTaskLogs">
              <RefreshCw :size="16" :class="{ spin: autoTaskLogsLoading }" />
              <span>{{ autoTaskLogsLoading ? '刷新中' : '刷新日志' }}</span>
            </button>
            <button class="table-action danger" type="button" :disabled="autoTaskLogsClearing || !autoTaskLogs.length" @click="clearAutoTaskLogs">
              <Trash2 :size="15" />
              <span>{{ autoTaskLogsClearing ? '清空中' : '一键清空' }}</span>
            </button>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>时间</th>
                  <th>任务</th>
                  <th>状态</th>
                  <th>扫描</th>
                  <th>成功</th>
                  <th>跳过</th>
                  <th>失败</th>
                  <th>文件</th>
                  <th>日志</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="log in autoTaskLogs" :key="log.id">
                  <td>{{ formatDateTime(log.createdAt) }}</td>
                  <td>{{ log.taskLabel || log.serverName || '-' }}</td>
                  <td>
                    <span class="status-pill" :class="statusClass(log.status)">
                      {{ statusLabel(log.status, log.taskType === 'SYNC' ? 'sync' : 'import') }}
                    </span>
                  </td>
                  <td>{{ formatNumber(log.scannedFiles) }}</td>
                  <td>{{ formatNumber(log.successFiles) }}</td>
                  <td>{{ formatNumber(log.skippedFiles) }}</td>
                  <td>{{ formatNumber(log.failedFiles) }}</td>
                  <td class="file-detail-cell" :title="log.fileDetails">{{ log.fileDetails || '-' }}</td>
                  <td class="message-cell">{{ log.message || '-' }}</td>
                </tr>
                <tr v-if="!autoTaskLogs.length">
                  <td colspan="9" class="empty">{{ autoTaskLogsLoading ? '正在加载自动任务日志...' : '暂无自动任务日志' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>
      </div>
      <div v-else class="empty-state"><p>{{ settingsLoading ? '加载中...' : '无法加载配置' }}</p></div>
    </section>

    <section v-else-if="currentView === 'profilePage'" class="result-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">PlayerLogger</p>
          <h1>个人中心</h1>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <span class="admin-badge">
            <UserRound :size="16" />
            {{ currentAdmin }}
          </span>
          <button class="secondary-button" type="button" @click="goDashboard">
            <ArrowLeft :size="18" />
            <span>返回统计</span>
          </button>
        </div>
      </div>

      <p v-if="error" class="notice error">{{ error }}</p>
      <p v-if="importMessage" class="notice success">{{ importMessage }}</p>

      <div class="settings-grid">
        <article class="panel wide">
          <div class="panel-title">
            <h2>账号信息</h2>
          </div>
          <div class="settings-form profile-form">
            <label>
              <span>账号</span>
              <input type="text" :value="currentAdmin" disabled />
            </label>
            <label>
              <span>登录有效期</span>
              <input type="text" :value="formatDateTime(currentAdminExpiresAt)" disabled />
            </label>
          </div>
        </article>

        <article class="panel wide">
          <div class="panel-title">
            <h2>修改密码</h2>
          </div>
          <form class="settings-form profile-form" @submit.prevent="changePassword">
            <label>
              <span>当前密码</span>
              <div class="password-field">
                <input :type="showCurrentPassword ? 'text' : 'password'" v-model="passwordForm.currentPassword" autocomplete="current-password" />
                <button type="button" class="icon-button ghost password-toggle" @click="showCurrentPassword = !showCurrentPassword" tabindex="-1">
                  <EyeOff v-if="showCurrentPassword" :size="16" />
                  <Eye v-else :size="16" />
                </button>
              </div>
            </label>
            <label>
              <span>新密码</span>
              <div class="password-field">
                <input :type="showNewPassword ? 'text' : 'password'" v-model="passwordForm.newPassword" autocomplete="new-password" />
                <button type="button" class="icon-button ghost password-toggle" @click="showNewPassword = !showNewPassword" tabindex="-1">
                  <EyeOff v-if="showNewPassword" :size="16" />
                  <Eye v-else :size="16" />
                </button>
              </div>
            </label>
            <label>
              <span>确认新密码</span>
              <div class="password-field">
                <input :type="showConfirmPassword ? 'text' : 'password'" v-model="passwordForm.confirmPassword" autocomplete="new-password" />
                <button type="button" class="icon-button ghost password-toggle" @click="showConfirmPassword = !showConfirmPassword" tabindex="-1">
                  <EyeOff v-if="showConfirmPassword" :size="16" />
                  <Eye v-else :size="16" />
                </button>
              </div>
            </label>
            <div class="settings-actions">
              <button class="primary-button" type="submit" :disabled="passwordSaving">
                <Save :size="18" />
                <span>{{ passwordSaving ? '保存中' : '保存密码' }}</span>
              </button>
            </div>
          </form>
        </article>
      </div>
    </section>

    <section v-else-if="currentView === 'syncPage'" class="result-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">PlayerLogger</p>
          <h1>CSV 复制</h1>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <button class="admin-badge" type="button" title="个人中心" @click="openProfilePage">
            <UserRound :size="16" />
            {{ currentAdmin }}
          </button>
          <button class="icon-button ghost" type="button" title="刷新文件对比" :disabled="syncPageLoading || syncJobRunning" @click="loadSyncPageFiles">
            <RefreshCw :size="18" :class="{ spin: syncPageLoading }" />
          </button>
          <label class="toggle-label">
            <input type="checkbox" v-model="syncSkipToday" :disabled="syncJobRunning" />
            <span>跳过当天文件</span>
          </label>
          <button class="primary-button" type="button" :disabled="syncingFiles || syncJobRunning" @click="startSyncJob">
            <Play :size="18" />
            <span>{{ syncJobRunning ? '复制中' : `开始复制 ${selectedServerName}` }}</span>
          </button>
          <button class="secondary-button" type="button" @click="goDashboard">
            <ArrowLeft :size="18" />
            <span>返回统计</span>
          </button>
        </div>
      </div>

      <p v-if="error" class="notice error">{{ error }}</p>
      <p v-if="importMessage" class="notice success">{{ importMessage }}</p>

      <section class="result-summary-grid">
        <article class="metric-card compact">
          <div class="metric-icon blue"><FileText :size="20" /></div>
          <p>扫描文件</p>
          <strong>{{ formatNumber(syncJob?.scannedFiles || 0) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon green"><CheckCircle2 :size="20" /></div>
          <p>已复制</p>
          <strong>{{ formatNumber(syncJob?.importedFiles || 0) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon amber"><CircleSlash :size="20" /></div>
          <p>跳过</p>
          <strong>{{ formatNumber(syncJob?.skippedFiles || 0) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon red"><XCircle :size="20" /></div>
          <p>失败</p>
          <strong>{{ formatNumber(syncJob?.failedFiles || 0) }}</strong>
        </article>
      </section>

      <article class="panel wide">
        <div class="panel-title">
          <h2>文件复制情况</h2>
          <span>{{ statusLabel(syncJob?.status || 'READY', 'sync') }}</span>
        </div>
        <div class="bulk-actions">
          <label class="bulk-check">
            <input
              type="checkbox"
              :checked="allLocalCopiesSelected"
              :disabled="syncJobRunning || !deletableLocalCopyFiles.length"
              @change="toggleAllLocalCopySelection($event.target.checked)"
            />
            <span>选择已在本地</span>
          </label>
          <button
            class="table-action danger"
            type="button"
            :disabled="syncJobRunning || deletingLocalCsv || !selectedLocalDeleteCount"
            @click="deleteSelectedLocalCsv"
          >
            <Trash2 :size="15" />
            <span>{{ deletingLocalCsv ? '删除中' : `删除本地 CSV ${selectedLocalDeleteCount}` }}</span>
          </button>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>选择</th>
                <th>文件</th>
                <th>服务器</th>
                <th>状态</th>
                <th>远程大小</th>
                <th>本地对比</th>
                <th>本地大小</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in syncPageFiles" :key="`${file.serverId}:${file.remotePath}`">
                <td>
                  <input
                    type="checkbox"
                    :checked="selectedLocalDeleteKeys.has(localCopyKey(file))"
                    :disabled="syncJobRunning || !canDeleteLocalCopy(file)"
                    @change="toggleLocalCopySelection(file, $event.target.checked)"
                  />
                </td>
                <td class="path-cell" :title="file.remotePath">{{ file.fileName || fileNameFromPath(file.remotePath) }}</td>
                <td>{{ file.serverName }}</td>
                <td>
                  <span class="status-pill" :class="statusClass(file.status)">
                    {{ statusLabel(file.status, 'sync') }}
                  </span>
                </td>
                <td>{{ fileSize(file.fileSize) }}</td>
                <td>
                  <span class="status-pill" :class="localCopyInfo(file).className" :title="localCopyInfo(file).message">
                    {{ localCopyInfo(file).label }}
                  </span>
                </td>
                <td>{{ localCopyInfo(file).fileSize ? fileSize(localCopyInfo(file).fileSize) : '-' }}</td>
                <td class="message-cell">{{ syncFileMessage(file) }}</td>
              </tr>
              <tr v-if="!syncPageFiles.length">
                <td colspan="8" class="empty">{{ syncPageLoading ? '正在加载文件对比...' : syncJobRunning ? '正在复制 CSV 文件' : '未找到远程文件，请检查 SMB 设置' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </section>

    <section v-else-if="currentView === 'importPage'" class="result-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">PlayerLogger</p>
          <h1>解析入库</h1>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <button class="admin-badge" type="button" title="个人中心" @click="openProfilePage">
            <UserRound :size="16" />
            {{ currentAdmin }}
          </button>
          <button class="icon-button ghost" type="button" title="刷新文件" :disabled="importPageLoading || importJobRunning" @click="loadImportFiles">
            <RefreshCw :size="18" :class="{ spin: importPageLoading }" />
          </button>
          <label class="toggle-label">
            <input type="checkbox" v-model="importSkipToday" :disabled="importJobRunning" />
            <span>跳过当天文件</span>
          </label>
          <button class="primary-button" type="button" :disabled="importing || importJobRunning || !importPageFiles.length" @click="startImportJob">
            <Play :size="18" />
            <span>{{ importJobRunning ? '解析中' : `开始解析 ${selectedServerName}` }}</span>
          </button>
          <button class="secondary-button" type="button" @click="goDashboard">
            <ArrowLeft :size="18" />
            <span>返回统计</span>
          </button>
        </div>
      </div>

      <p v-if="error" class="notice error">{{ error }}</p>
      <p v-if="importMessage" class="notice success">{{ importMessage }}</p>

      <section class="result-summary-grid">
        <article class="metric-card compact">
          <div class="metric-icon blue"><FileText :size="20" /></div>
          <p>本地 CSV</p>
          <strong>{{ formatNumber(importPageFiles.length) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon green"><CheckCircle2 :size="20" /></div>
          <p>已导入</p>
          <strong>{{ formatNumber(importJob?.importedFiles || importFiles.filter((file) => file.status === 'IMPORTED').length) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon amber"><CircleSlash :size="20" /></div>
          <p>跳过</p>
          <strong>{{ formatNumber(importJob?.skippedFiles || importFiles.filter((file) => file.status === 'SKIPPED_TODAY').length) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon red"><XCircle :size="20" /></div>
          <p>失败</p>
          <strong>{{ formatNumber(importJob?.failedFiles || 0) }}</strong>
        </article>
      </section>

      <article class="panel wide">
        <div class="panel-title">
          <h2>文件解析情况</h2>
          <span>{{ importJob?.status || 'READY' }}</span>
        </div>
        <div class="bulk-actions">
          <label class="bulk-check">
            <input
              type="checkbox"
              :checked="allEligibleSelected"
              :disabled="importJobRunning || !eligibleImportFiles.length"
              @change="toggleAllImportFileSelection($event.target.checked)"
            />
            <span>选择可删除记录</span>
          </label>
          <button
            class="table-action danger"
            type="button"
            :disabled="importJobRunning || resetRemotePath === '__batch__' || !selectedResetCount"
            @click="deleteSelectedImportRecords"
          >
            <Trash2 :size="15" />
            <span>{{ resetRemotePath === '__batch__' ? '批量删除中' : `批量删记录 ${selectedResetCount}` }}</span>
          </button>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>选择</th>
                <th>文件</th>
                <th>服务器</th>
                <th>状态</th>
                <th>大小</th>
                <th>读取行</th>
                <th>忽略行</th>
                <th>说明</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in importPageFiles" :key="`${file.serverId}:${file.remotePath}`">
                <td>
                  <input
                    type="checkbox"
                    :checked="selectedResetKeys.has(importFileKey(file))"
                    :disabled="importJobRunning || !canDeleteImportRecord(file)"
                    @change="toggleImportFileSelection(file, $event.target.checked)"
                  />
                </td>
                <td class="path-cell" :title="file.remotePath">{{ file.fileName || fileNameFromPath(file.remotePath) }}</td>
                <td>{{ file.serverName }}</td>
                <td>
                  <span class="status-pill" :class="statusClass(file.status)">
                    {{ statusLabel(file.status) }}
                  </span>
                </td>
                <td>{{ fileSize(file.fileSize) }}</td>
                <td>{{ formatNumber(file.rowCount) }}</td>
                <td>{{ formatNumber(file.ignoredCount) }}</td>
                <td class="message-cell">{{ file.message || '-' }}</td>
                <td>
                  <button
                    class="table-action danger"
                    type="button"
                    :disabled="importJobRunning || resetRemotePath === file.remotePath || !canDeleteImportRecord(file)"
                    @click="deleteImportRecord(file)"
                  >
                    <Trash2 :size="15" />
                    <span>{{ resetRemotePath === file.remotePath ? '删除中' : '删记录' }}</span>
                  </button>
                </td>
              </tr>
              <tr v-if="!importPageFiles.length">
                <td colspan="9" class="empty">{{ importPageLoading ? '正在扫描本地 CSV' : '暂无本地 CSV 文件' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </section>

    <section v-else-if="currentView === 'logQueryPage'" class="result-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">PlayerLogger</p>
          <h1>查日志</h1>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <button class="admin-badge" type="button" title="个人中心" @click="openProfilePage">
            <UserRound :size="16" />
            {{ currentAdmin }}
          </button>
          <button class="icon-button ghost" type="button" title="刷新查询状态" :disabled="selectedLogQueryLoading" @click="loadLogFeatureState(logQueryServerId)">
            <RefreshCw :size="18" :class="{ spin: selectedLogQueryLoading || selectedLogQueryRunning }" />
          </button>
          <button class="secondary-button" type="button" @click="goDashboard">
            <ArrowLeft :size="18" />
            <span>返回统计</span>
          </button>
        </div>
      </div>

      <p v-if="error" class="notice error">{{ error }}</p>
      <p v-if="importMessage" class="notice success">{{ importMessage }}</p>

      <nav class="log-feature-tabs" aria-label="日志功能">
        <button
          type="button"
          :class="{ active: logQueryMode === LOG_QUERY_TYPE_COORDINATE }"
          :aria-current="logQueryMode === LOG_QUERY_TYPE_COORDINATE ? 'page' : undefined"
          @click="setLogQueryMode(LOG_QUERY_TYPE_COORDINATE)"
        >
          <Crosshair :size="16" />
          <span>通过坐标查询</span>
        </button>
        <button
          type="button"
          :class="{ active: logQueryMode === LOG_QUERY_TYPE_PLAYER_KEYWORD }"
          :aria-current="logQueryMode === LOG_QUERY_TYPE_PLAYER_KEYWORD ? 'page' : undefined"
          @click="setLogQueryMode(LOG_QUERY_TYPE_PLAYER_KEYWORD)"
        >
          <Search :size="16" />
          <span>综合筛选</span>
        </button>
        <button
          type="button"
          :class="{ active: logQueryMode === LOG_QUERY_TYPE_XRAY }"
          :aria-current="logQueryMode === LOG_QUERY_TYPE_XRAY ? 'page' : undefined"
          @click="setLogQueryMode(LOG_QUERY_TYPE_XRAY)"
        >
          <Eye :size="16" />
          <span>矿透分析</span>
        </button>
      </nav>

      <section class="server-tabs compact" aria-label="日志查询服务器">
        <button
          v-for="server in serverOptions"
          :key="server.serverId"
          type="button"
          :class="{ active: logQueryServerId === server.serverId }"
          @click="setLogQueryServer(server.serverId)"
        >
          <Search :size="16" />
          <span>{{ server.serverName }}</span>
          <small v-if="isActiveJob(logFeatureResultFor(server.serverId))">查询中</small>
          <small v-else-if="logQueryMode === LOG_QUERY_TYPE_XRAY && xrayAnalysisResultFor(server.serverId)?.maxRiskScore">
            {{ xrayAnalysisResultFor(server.serverId).maxRiskScore }}分
          </small>
          <small v-else-if="logFeatureResultFor(server.serverId)?.matchedRows">{{ formatNumber(logFeatureResultFor(server.serverId).matchedRows) }}</small>
        </button>
      </section>

      <article class="panel wide">
        <div class="panel-title">
          <h2>{{ serverNameById(logQueryServerId) }} {{ logQueryModeLabel(logQueryMode) }}</h2>
          <span>{{ logQueryStatusLabel(selectedLogFeatureResult?.status) }}</span>
        </div>

        <form v-if="logQueryMode === LOG_QUERY_TYPE_COORDINATE" class="log-query-form" @submit.prevent="startLogQuery(logQueryServerId)">
          <label>
            <span>开始日期</span>
            <input v-model="logQueryForms[logQueryServerId].fromDate" type="date" required @change="saveLogQueryForms" />
          </label>
          <label>
            <span>结束日期</span>
            <input v-model="logQueryForms[logQueryServerId].toDate" type="date" required @change="saveLogQueryForms" />
          </label>
          <label>
            <span>维度</span>
            <input v-model.trim="logQueryForms[logQueryServerId].dimension" type="text" placeholder="主世界/下界/末地，留空不限" @change="saveLogQueryForms" />
          </label>
          <div class="coordinate-group">
            <span>交互坐标范围起点</span>
            <input v-model="logQueryForms[logQueryServerId].x1" type="number" step="any" placeholder="X" required @change="saveLogQueryForms" />
            <input v-model="logQueryForms[logQueryServerId].y1" type="number" step="any" placeholder="Y" required @change="saveLogQueryForms" />
            <input v-model="logQueryForms[logQueryServerId].z1" type="number" step="any" placeholder="Z" required @change="saveLogQueryForms" />
          </div>
          <div class="coordinate-group">
            <span>交互坐标范围终点</span>
            <input v-model="logQueryForms[logQueryServerId].x2" type="number" step="any" placeholder="X" required @change="saveLogQueryForms" />
            <input v-model="logQueryForms[logQueryServerId].y2" type="number" step="any" placeholder="Y" required @change="saveLogQueryForms" />
            <input v-model="logQueryForms[logQueryServerId].z2" type="number" step="any" placeholder="Z" required @change="saveLogQueryForms" />
          </div>
          <div class="log-query-actions">
            <button class="primary-button" type="submit" :disabled="selectedLogQueryRunning || selectedLogQueryLoading">
              <Search :size="18" />
              <span>{{ selectedLogQueryRunning ? '查询中' : `开始查询 ${serverNameById(logQueryServerId)}` }}</span>
            </button>
            <button class="secondary-button" type="button" :disabled="selectedLogQueryRunning" @click="clearLogQuery(logQueryServerId)">
              <Trash2 :size="18" />
              <span>清空结果</span>
            </button>
          </div>
        </form>

        <form v-else-if="logQueryMode === LOG_QUERY_TYPE_PLAYER_KEYWORD" class="log-query-form" @submit.prevent="startLogQuery(logQueryServerId)">
          <label>
            <span>开始日期（留空不限）</span>
            <input v-model="logTextQueryForms[logQueryServerId].fromDate" type="date" @change="saveLogTextQueryForms" />
          </label>
          <label>
            <span>结束日期（留空不限）</span>
            <input v-model="logTextQueryForms[logQueryServerId].toDate" type="date" @change="saveLogTextQueryForms" />
          </label>
          <label>
            <span>玩家名</span>
            <input v-model.trim="logTextQueryForms[logQueryServerId].playerName" type="text" placeholder="玩家名" @change="saveLogTextQueryForms" />
          </label>
          <label>
            <span>关键字</span>
            <input v-model.trim="logTextQueryForms[logQueryServerId].keyword" type="text" placeholder="详细信息关键字" @change="saveLogTextQueryForms" />
          </label>
          <label>
            <span>行为</span>
            <input v-model.trim="logTextQueryForms[logQueryServerId].action" type="text" placeholder="行为筛选" @change="saveLogTextQueryForms" />
          </label>
          <div class="log-query-actions">
            <button class="primary-button" type="submit" :disabled="selectedLogQueryRunning || selectedLogQueryLoading">
              <Search :size="18" />
              <span>{{ selectedLogQueryRunning ? '查询中' : `开始查询 ${serverNameById(logQueryServerId)}` }}</span>
            </button>
            <button class="secondary-button" type="button" :disabled="selectedLogQueryRunning" @click="clearLogQuery(logQueryServerId)">
              <Trash2 :size="18" />
              <span>清空结果</span>
            </button>
          </div>
        </form>

        <form v-else class="log-query-form" @submit.prevent="startLogQuery(logQueryServerId)">
          <label>
            <span>开始日期</span>
            <input v-model="xrayAnalysisForms[logQueryServerId].fromDate" type="date" required @change="saveXrayAnalysisForms" />
          </label>
          <label>
            <span>结束日期</span>
            <input v-model="xrayAnalysisForms[logQueryServerId].toDate" type="date" required @change="saveXrayAnalysisForms" />
          </label>
          <label>
            <span>玩家名</span>
            <input v-model.trim="xrayAnalysisForms[logQueryServerId].playerName" type="text" placeholder="留空分析全部玩家" @change="saveXrayAnalysisForms" />
          </label>
          <label>
            <span>维度</span>
            <input v-model.trim="xrayAnalysisForms[logQueryServerId].dimension" type="text" placeholder="主世界/下界，留空不限" @change="saveXrayAnalysisForms" />
          </label>
          <div class="log-query-actions">
            <button class="primary-button" type="submit" :disabled="selectedLogQueryRunning || selectedLogQueryLoading">
              <Eye :size="18" />
              <span>{{ selectedLogQueryRunning ? '分析中' : `开始分析 ${serverNameById(logQueryServerId)}` }}</span>
            </button>
            <button class="secondary-button" type="button" :disabled="selectedLogQueryRunning" @click="clearLogQuery(logQueryServerId)">
              <Trash2 :size="18" />
              <span>清空结果</span>
            </button>
          </div>
        </form>
      </article>

      <section class="result-summary-grid">
        <article class="metric-card compact">
          <div class="metric-icon blue"><FileText :size="20" /></div>
          <p>扫描文件</p>
          <strong>{{ formatNumber(logQueryMode === LOG_QUERY_TYPE_XRAY ? selectedXrayAnalysisResult?.scannedFiles || 0 : selectedLogQueryResult?.scannedFiles || 0) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon green"><Search :size="20" /></div>
          <p>{{ logQueryMode === LOG_QUERY_TYPE_XRAY ? '分析玩家' : '匹配事件' }}</p>
          <strong>{{ formatNumber(logQueryMode === LOG_QUERY_TYPE_XRAY ? selectedXrayAnalysisResult?.playerCount || 0 : selectedLogQueryResult?.matchedRows || 0) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon amber"><Database :size="20" /></div>
          <p>{{ logQueryMode === LOG_QUERY_TYPE_XRAY ? '最高风险' : '扫描行数' }}</p>
          <strong>{{ formatNumber(logQueryMode === LOG_QUERY_TYPE_XRAY ? selectedXrayAnalysisResult?.maxRiskScore || 0 : selectedLogQueryResult?.scannedRows || 0) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon red"><XCircle :size="20" /></div>
          <p>解析失败</p>
          <strong>{{ formatNumber(logQueryMode === LOG_QUERY_TYPE_XRAY ? selectedXrayAnalysisResult?.failedFiles || 0 : selectedLogQueryResult?.failedFiles || 0) }}</strong>
        </article>
      </section>

      <article class="panel wide">
        <div class="panel-title">
          <h2>{{ logQueryMode === LOG_QUERY_TYPE_XRAY ? '分析结果' : '查询结果' }}</h2>
          <span>{{ selectedLogFeatureResult?.message || '还没有查询记录' }}</span>
        </div>
        <p v-if="selectedLogQueryRunning" class="settings-hint">
          正在扫描 {{ selectedLogFeatureResult?.currentFile || 'CSV 文件' }}，关掉页面也可以稍后回来查看结果。
        </p>
        <template v-if="logQueryMode === LOG_QUERY_TYPE_XRAY">
          <div class="table-toolbar">
            <span>
              疑似玩家 {{ formatNumber(selectedXrayAnalysisResult?.playerCount || 0) }} 名，风险项 {{ formatNumber(selectedXrayAnalysisResult?.findingCount || 0) }} 个，最高 {{ formatNumber(selectedXrayAnalysisResult?.maxRiskScore || 0) }} 分
            </span>
          </div>
          <div class="table-wrap">
            <table class="xray-table">
              <thead>
                <tr>
                  <th>玩家</th>
                  <th>疑似风险</th>
                  <th>周期稀有矿</th>
                  <th>周期十分钟峰值</th>
                  <th>追矿证据</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="player in (selectedXrayAnalysisResult?.players || [])" :key="player.playerName">
                  <td>{{ player.playerName }}</td>
                  <td>
                    <span class="risk-pill" :class="xrayRiskClass(player.riskScore)">
                      {{ player.riskScore }} / 100
                    </span>
                    <small class="risk-level">{{ player.riskLevel }}</small>
                  </td>
                  <td>
                    <div class="ore-stack compact">
                      <strong>{{ formatNumber(xrayAnalysisValue(player, 'analysisRareOreBreaks', 'miningSessionRareOreBreaks')) }}</strong>
                    </div>
                  </td>
                  <td>
                    <div class="ore-stack compact">
                      <strong>{{ formatNumber(xrayAnalysisValue(player, 'analysisPeakRareOreWindowCount', 'peakRareOreWindowCount')) }}</strong>
                    </div>
                  </td>
                  <td>{{ formatNumber(xrayAnalysisTrackingEvidenceCount(player)) }}</td>
                  <td>
                    <button class="secondary-button compact-button" type="button" @click="openXrayPlayerDetail(player)">
                      <Eye :size="16" />
                      <span>查看详细</span>
                    </button>
                  </td>
                </tr>
                <tr v-if="!(selectedXrayAnalysisResult?.players || []).length">
                  <td colspan="6" class="empty">
                    {{ selectedLogQueryRunning ? '正在分析，请稍等' : (selectedXrayAnalysisResult?.message || '还没有分析记录') }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
        <template v-else>
        <div class="table-toolbar">
          <span>
            第 {{ selectedLogQueryPage }} / {{ selectedLogQueryTotalPages || 1 }} 页，本页 {{ formatNumber(selectedLogQueryRows.length) }} 条，共 {{ formatNumber(selectedLogQueryResult?.matchedRows || 0) }} 条
          </span>
          <div class="pagination-controls">
            <label>
              <span>每页</span>
              <select :value="selectedLogQueryPageSize" @change="setLogQueryPageSize($event.target.value)">
                <option v-for="size in LOG_QUERY_PAGE_SIZE_OPTIONS" :key="size" :value="size">{{ size }}</option>
              </select>
            </label>
            <button
              class="icon-button"
              type="button"
              title="上一页"
              :disabled="selectedLogQueryPage <= 1 || selectedLogQueryLoading"
              @click="setLogQueryPage(selectedLogQueryPage - 1)"
            >
              <ChevronLeft :size="18" />
            </button>
            <button
              class="icon-button"
              type="button"
              title="下一页"
              :disabled="selectedLogQueryPage >= (selectedLogQueryTotalPages || 1) || selectedLogQueryLoading"
              @click="setLogQueryPage(selectedLogQueryPage + 1)"
            >
              <ChevronRight :size="18" />
            </button>
          </div>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>文件</th>
                <th>行号</th>
                <th>时间</th>
                <th>玩家</th>
                <th>行为</th>
                <th>玩家坐标</th>
                <th>交互坐标</th>
                <th>详细1</th>
                <th>详细2</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in selectedLogQueryRows" :key="`${row.filePath}:${row.lineNumber}`">
                <td class="path-cell" :title="row.filePath">{{ row.fileName }}</td>
                <td>{{ row.lineNumber }}</td>
                <td>{{ row.date }} {{ row.time }}</td>
                <td>{{ row.playerName }}</td>
                <td>{{ row.action }}</td>
                <td>{{ logQueryCoordinateText(row.x, row.y, row.z, row.dimension) }}</td>
                <td>{{ logQueryCoordinateText(row.x2, row.y2, row.z2, row.dimension2) }}</td>
                <td class="message-cell">{{ row.detail1 || '-' }}</td>
                <td class="message-cell">{{ row.detail2 || '-' }}</td>
              </tr>
              <tr v-if="!selectedLogQueryRows.length">
                <td colspan="9" class="empty">
                  {{ selectedLogQueryRunning ? '正在查询，请稍等' : (selectedLogQueryResult?.message || '还没有查询记录') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="table-toolbar bottom-toolbar">
          <span>
            第 {{ selectedLogQueryPage }} / {{ selectedLogQueryTotalPages || 1 }} 页，本页 {{ formatNumber(selectedLogQueryRows.length) }} 条，共 {{ formatNumber(selectedLogQueryResult?.matchedRows || 0) }} 条
          </span>
          <div class="pagination-controls">
            <label>
              <span>每页</span>
              <select :value="selectedLogQueryPageSize" @change="setLogQueryPageSize($event.target.value)">
                <option v-for="size in LOG_QUERY_PAGE_SIZE_OPTIONS" :key="size" :value="size">{{ size }}</option>
              </select>
            </label>
            <button
              class="icon-button"
              type="button"
              title="上一页"
              :disabled="selectedLogQueryPage <= 1 || selectedLogQueryLoading"
              @click="setLogQueryPage(selectedLogQueryPage - 1)"
            >
              <ChevronLeft :size="18" />
            </button>
            <button
              class="icon-button"
              type="button"
              title="下一页"
              :disabled="selectedLogQueryPage >= (selectedLogQueryTotalPages || 1) || selectedLogQueryLoading"
              @click="setLogQueryPage(selectedLogQueryPage + 1)"
            >
              <ChevronRight :size="18" />
            </button>
            <button class="secondary-button" type="button" @click="scrollLogQueryToTop">
              <ArrowUp :size="18" />
              <span>返回顶部</span>
            </button>
          </div>
        </div>
        </template>
      </article>
    </section>

    <section v-else-if="currentView === 'xrayDetailPage'" class="result-page xray-detail-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">矿透分析详情</p>
          <h1>{{ selectedXrayDetailPlayer?.playerName || xrayDetailPlayerName || '玩家详情' }}</h1>
          <p class="muted-line">{{ serverNameById(logQueryServerId) }}</p>
          <p class="muted-line">统计日期：{{ xrayAnalysisDateRange(xrayDetailResult) }}</p>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <button class="primary-button" type="button" :disabled="!selectedXrayDetailPlayer" @click="openXrayGroupSendDialog">
            <Send :size="18" />
            <span>发送到Q群</span>
          </button>
          <button class="secondary-button" type="button" @click="backToXrayAnalysis">
            <ArrowLeft :size="18" />
            <span>返回矿透分析</span>
          </button>
        </div>
      </div>

      <article v-if="selectedLogQueryLoading" class="panel wide">
        <p class="settings-hint">正在读取矿透分析详情...</p>
      </article>

      <article v-else-if="!selectedXrayDetailPlayer" class="panel wide">
        <div class="panel-title">
          <h2>没有找到这个玩家</h2>
          <span>{{ xrayDetailPlayerName || '-' }}</span>
        </div>
        <p class="settings-hint">可能还没有分析结果，或者最新结果里没有这个玩家。返回矿透分析重新运行一次即可。</p>
      </article>

      <template v-else>
        <article class="panel wide">
          <div class="xray-detail-header">
            <div>
              <h3>{{ selectedXrayDetailPlayer.playerName }}</h3>
              <span>{{ selectedXrayDetailPlayer.riskScore }} / 100 · {{ selectedXrayDetailPlayer.riskLevel }}</span>
            </div>
            <span class="risk-pill" :class="xrayRiskClass(selectedXrayDetailPlayer.riskScore)">
              {{ selectedXrayDetailPlayer.riskScore }} / 100
            </span>
          </div>

          <div class="xray-detail-body">
            <section class="xray-detail-section xray-toggle-section">
              <h4>最可疑挖矿会话</h4>
              <div class="xray-detail-grid">
                <div>
                  <span>会话时间</span>
                  <strong>{{ xrayMiningSessionTime(selectedXrayDetailPlayer) }}</strong>
                </div>
                <div>
                  <span>会话时长</span>
                  <strong>{{ xrayMiningSessionDuration(selectedXrayDetailPlayer) }}</strong>
                </div>
                <div>
                  <span>会话地下破坏</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.miningSessionUndergroundBreaks || selectedXrayDetailPlayer.miningSessionBreaks || 0) }}</strong>
                </div>
                <div>
                  <span>地下稀有矿占比</span>
                  <strong>{{ xrayRareRatioText(selectedXrayDetailPlayer) }}</strong>
                </div>
                <div>
                  <span>稀有矿脉</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.miningSessionRareVeins || 0) }}</strong>
                </div>
                <div>
                  <span>十分钟矿脉峰值</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.peakRareVeinWindowCount || 0) }}</strong>
                </div>
              </div>
            </section>

            <section class="xray-detail-section xray-toggle-section">
              <h4>筛选周期总量</h4>
              <div class="xray-detail-grid">
                <div>
                  <span>破坏方块</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisBreaks', 'miningSessionBreaks')) }}</strong>
                </div>
                <div>
                  <span>地下破坏</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisUndergroundBreaks', 'miningSessionUndergroundBreaks')) }}</strong>
                </div>
                <div>
                  <span>稀有矿</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisRareOreBreaks', 'miningSessionRareOreBreaks')) }}</strong>
                </div>
                <div>
                  <span>钻石矿</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisDiamondOreBreaks', 'miningSessionDiamondOreBreaks')) }}</strong>
                </div>
                <div>
                  <span>远古残骸</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisAncientDebrisBreaks', 'miningSessionAncientDebrisBreaks')) }}</strong>
                </div>
                <div>
                  <span>地下稀有矿占比</span>
                  <strong>{{ xrayAnalysisRareRatioText(selectedXrayDetailPlayer) }}</strong>
                </div>
                <div>
                  <span>稀有矿脉</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisRareVeins', 'miningSessionRareVeins')) }}</strong>
                </div>
                <div>
                  <span>十分钟挖取峰值</span>
                  <strong>{{ formatNumber(xrayAnalysisValue(selectedXrayDetailPlayer, 'analysisPeakRareOreWindowCount', 'peakRareOreWindowCount')) }}</strong>
                </div>
                <div>
                  <span>周期追矿证据</span>
                  <strong>{{ formatNumber(xrayAnalysisTrackingEvidenceCount(selectedXrayDetailPlayer)) }}</strong>
                </div>
              </div>
            </section>

            <section class="xray-detail-section xray-toggle-section">
              <h4>会话风险概况</h4>
              <div class="xray-detail-grid">
                <div>
                  <span>稀有矿</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.miningSessionRareOreBreaks) }}</strong>
                </div>
                <div>
                  <span>钻石矿</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.miningSessionDiamondOreBreaks) }}</strong>
                </div>
                <div>
                  <span>远古残骸</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.miningSessionAncientDebrisBreaks) }}</strong>
                </div>
                <div>
                  <span>会话追矿证据</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.trackingEvidenceCount) }}</strong>
                </div>
                <div>
                  <span>十分钟挖取峰值</span>
                  <strong>{{ formatNumber(selectedXrayDetailPlayer.peakRareOreWindowCount || 0) }}</strong>
                </div>
                <div>
                  <span>短时窗口</span>
                  <strong v-if="selectedXrayDetailPlayer.peakRareOreWindowStart && selectedXrayDetailPlayer.peakRareOreWindowEnd">
                    {{ formatDateTime(selectedXrayDetailPlayer.peakRareOreWindowStart) }} - {{ formatDateTime(selectedXrayDetailPlayer.peakRareOreWindowEnd) }}
                  </strong>
                  <strong v-else>-</strong>
                </div>
              </div>
            </section>

            <section class="xray-detail-section xray-toggle-section">
              <h4>原因</h4>
              <div class="xray-chip-list">
                <span v-for="reason in selectedXrayDetailPlayer.reasons" :key="reason">{{ reason }}</span>
                <span v-if="!selectedXrayDetailPlayer.reasons?.length">-</span>
              </div>
            </section>

            <section class="xray-detail-section">
              <h4>筛选周期矿物明细</h4>
              <div class="xray-chip-list">
                <span v-for="ore in xrayAnalysisOreCounts(selectedXrayDetailPlayer)" :key="ore.oreType">
                  {{ ore.displayName }} x{{ formatNumber(ore.count) }}
                </span>
                <span v-if="!xrayAnalysisOreCounts(selectedXrayDetailPlayer).length">-</span>
              </div>
            </section>

            <section class="xray-detail-section">
              <h4>筛选周期证据</h4>
              <div class="xray-expand-row">
                <button
                  v-if="selectedXrayDetailPlayer.evidence?.length"
                  class="secondary-button mini-button"
                  type="button"
                  @click="xrayDetailEvidenceExpanded = !xrayDetailEvidenceExpanded"
                >
                  <span>{{ xrayDetailEvidenceExpanded ? '收起' : `展开 ${selectedXrayDetailPlayer.evidence.length} 条` }}</span>
                </button>
              </div>
              <div v-if="selectedXrayDetailPlayer.evidence?.length && xrayDetailEvidenceExpanded" class="xray-evidence-list">
                <div v-for="(evidence, evidenceIndex) in selectedXrayDetailPlayer.evidence" :key="`${selectedXrayDetailPlayer.playerName}:${evidence.startedAt}:${evidence.summary}`" class="evidence-block">
                  <strong>{{ evidence.summary }}</strong>
                  <small>{{ xrayEvidenceTime(evidence) }} · +{{ evidence.score }}</small>
                  <button
                    v-if="evidence.rows?.length"
                    class="secondary-button mini-button"
                    type="button"
                    @click="toggleDetailXrayEvidenceRows(evidence, evidenceIndex)"
                  >
                    <span>{{ isDetailXrayEvidenceRowsExpanded(evidence, evidenceIndex) ? '收起明细' : `展开明细 ${evidence.rows.length} 条` }}</span>
                  </button>
                  <div v-if="evidence.rows?.length && isDetailXrayEvidenceRowsExpanded(evidence, evidenceIndex)" class="evidence-lines">
                    <span v-for="row in evidence.rows" :key="`${row.filePath}:${row.lineNumber}`">
                      {{ row.time }} {{ row.detail1 || '-' }} @ {{ logQueryCoordinateText(row.x2, row.y2, row.z2, row.dimension2) }}
                    </span>
                  </div>
                </div>
              </div>
              <p v-else-if="!selectedXrayDetailPlayer.evidence?.length" class="settings-hint">没有追矿证据。</p>
            </section>

            <section class="xray-detail-section">
              <h4>稀有矿位置</h4>
              <div class="xray-expand-row">
                <button
                  v-if="selectedXrayDetailPlayer.rareOreRows?.length"
                  class="secondary-button mini-button"
                  type="button"
                  @click="toggleXrayDetailOrePositions"
                >
                  <span>{{ xrayDetailOrePositionsExpanded ? '收起' : `展开 ${xrayDetailRareOreRows.length} 条` }}</span>
                </button>
              </div>
              <div v-if="selectedXrayDetailPlayer.rareOreRows?.length && xrayDetailOrePositionsExpanded" class="table-wrap">
                <div class="table-toolbar">
                  <span>
                    第 {{ clampPage(xrayDetailOrePositionPage, xrayDetailOrePositionTotalPages) }} / {{ xrayDetailOrePositionTotalPages }} 页，本页 {{ formatNumber(xrayDetailPagedRareOreRows.length) }} 条，共 {{ formatNumber(xrayDetailRareOreRows.length) }} 条
                  </span>
                  <div class="pagination-controls">
                    <label>
                      <span>每页</span>
                      <select :value="xrayDetailOrePositionPageSize" @change="setXrayDetailOrePositionPageSize($event.target.value)">
                        <option v-for="size in XRAY_ORE_POSITION_PAGE_SIZE_OPTIONS" :key="size" :value="size">{{ size }}</option>
                      </select>
                    </label>
                    <button class="icon-button" type="button" title="上一页" :disabled="xrayDetailOrePositionPage <= 1" @click="setXrayDetailOrePositionPage(xrayDetailOrePositionPage - 1)">
                      <ChevronLeft :size="18" />
                    </button>
                    <button class="icon-button" type="button" title="下一页" :disabled="xrayDetailOrePositionPage >= xrayDetailOrePositionTotalPages" @click="setXrayDetailOrePositionPage(xrayDetailOrePositionPage + 1)">
                      <ChevronRight :size="18" />
                    </button>
                  </div>
                </div>
                <table class="xray-ore-position-table">
                  <thead>
                    <tr>
                      <th>顺序</th>
                      <th>时间</th>
                      <th>矿物</th>
                      <th>玩家坐标</th>
                      <th>交互坐标</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, index) in xrayDetailPagedRareOreRows" :key="`${row.filePath}:${row.lineNumber}:rare`">
                      <td>{{ xrayDetailOrePositionOffset + index + 1 }}</td>
                      <td>{{ row.date }} {{ row.time }}</td>
                      <td class="message-cell">{{ rareOreDetailText(row) }}</td>
                      <td>{{ logQueryCoordinateText(row.x, row.y, row.z, row.dimension) }}</td>
                      <td>{{ logQueryCoordinateText(row.x2, row.y2, row.z2, row.dimension2) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p v-else-if="!selectedXrayDetailPlayer.rareOreRows?.length" class="settings-hint">没有稀有矿位置记录。</p>
            </section>

            <div class="xray-detail-actions">
              <button class="secondary-button" type="button" @click="scrollLogQueryToTop">
                <ArrowUp :size="18" />
                <span>返回顶部</span>
              </button>
            </div>
          </div>
        </article>
      </template>

      <div v-if="xrayGroupSendDialogOpen" class="modal-backdrop" @click.self="closeXrayGroupSendDialog">
        <form class="modal-panel" @submit.prevent="sendXrayDetailToGroup">
          <div class="modal-title">
            <div>
              <p class="eyebrow">AstrBot</p>
              <h2>发送到Q群</h2>
            </div>
            <button class="icon-button ghost" type="button" title="关闭" @click="closeXrayGroupSendDialog">
              <XCircle :size="18" />
            </button>
          </div>

          <label>
            <span>链接有效时间</span>
            <input v-model.trim="xrayGroupSendTtlText" type="text" placeholder="留空默认 1天，可填 6小时/30分钟/2天" />
          </label>
          <p class="settings-hint">发送内容使用 AstrBot 插件的 share_base_url 生成详情链接；只填数字时按天计算。</p>
          <p v-if="xrayGroupSendError" class="notice error">{{ xrayGroupSendError }}</p>
          <p v-if="xrayGroupSendMessage" class="notice success">{{ xrayGroupSendMessage }}</p>

          <div class="modal-actions">
            <button class="secondary-button" type="button" @click="closeXrayGroupSendDialog">取消</button>
            <button class="primary-button" type="submit" :disabled="xrayGroupSendLoading">
              <Send :size="18" />
              <span>{{ xrayGroupSendLoading ? '提交中...' : '发送到Q群' }}</span>
            </button>
          </div>
        </form>
      </div>
    </section>

    <section v-else-if="currentView === 'operationResult'" class="result-page">
      <div class="topbar">
        <div>
          <p class="eyebrow">PlayerLogger</p>
          <h1>{{ lastOperation?.title || '操作结果' }}</h1>
        </div>
        <div class="actions">
          <span v-if="serverTimeDisplay" class="server-time-badge">
            <Clock :size="16" />
            {{ serverTimeDisplay }}
          </span>
          <button class="admin-badge" type="button" title="个人中心" @click="openProfilePage">
            <UserRound :size="16" />
            {{ currentAdmin }}
          </button>
          <button class="secondary-button" type="button" @click="goDashboard">
            <ArrowLeft :size="18" />
            <span>返回统计</span>
          </button>
          <button class="icon-button ghost" type="button" title="退出登录" @click="logout">
            <LogOut :size="18" />
          </button>
        </div>
      </div>

      <section class="result-summary-grid">
        <article class="metric-card compact">
          <div class="metric-icon blue"><FileText :size="20" /></div>
          <p>扫描文件</p>
          <strong>{{ formatNumber(lastOperation?.result?.scannedFiles) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon green"><CheckCircle2 :size="20" /></div>
          <p>{{ lastOperation?.actionLabel || '成功' }}</p>
          <strong>{{ formatNumber(lastOperation?.result?.importedFiles) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon amber"><CircleSlash :size="20" /></div>
          <p>跳过</p>
          <strong>{{ formatNumber(lastOperation?.result?.skippedFiles) }}</strong>
        </article>
        <article class="metric-card compact">
          <div class="metric-icon red"><XCircle :size="20" /></div>
          <p>失败</p>
          <strong>{{ formatNumber(lastOperation?.result?.failedFiles) }}</strong>
        </article>
      </section>

      <article class="panel wide">
        <div class="panel-title">
          <h2>文件明细</h2>
          <span>{{ operationFiles.length }} 个文件</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>文件</th>
                <th>服务器</th>
                <th>状态</th>
                <th>读取行</th>
                <th>忽略行</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in operationFiles" :key="`${file.serverId}:${file.remotePath}`">
                <td class="path-cell" :title="file.remotePath">{{ fileNameFromPath(file.remotePath) }}</td>
                <td>{{ file.serverName }}</td>
                <td>
                  <span class="status-pill" :class="statusClass(file.status)">
                    {{ statusLabel(file.status) }}
                  </span>
                </td>
                <td>{{ formatNumber(file.rowCount) }}</td>
                <td>{{ formatNumber(file.ignoredCount) }}</td>
                <td class="message-cell">{{ file.message || '-' }}</td>
              </tr>
              <tr v-if="!operationFiles.length">
                <td colspan="6" class="empty">暂无文件结果</td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </section>

    <template v-else>
    <section class="topbar">
      <div>
        <p class="eyebrow">PlayerLogger</p>
        <h1>玩家日志统计</h1>
      </div>
      <div class="actions">
        <span v-if="serverTimeDisplay" class="server-time-badge">
          <Clock :size="16" />
          {{ serverTimeDisplay }}
        </span>
        <button class="admin-badge" type="button" title="个人中心" @click="openProfilePage">
          <UserRound :size="16" />
          {{ currentAdmin }}
        </button>
        <button class="icon-button ghost" type="button" title="刷新" :disabled="loading" @click="loadAll">
          <RefreshCw :size="18" :class="{ spin: loading }" />
        </button>
        <button v-if="filters.serverId !== 'all'" class="primary-button" type="button" @click="openSyncPage">
          <Download :size="18" />
          <span>{{ syncNavLabel }}</span>
        </button>
        <button v-if="filters.serverId !== 'all'" class="secondary-button" type="button" @click="openImportPage">
          <Database :size="18" />
          <span>{{ importNavLabel }}</span>
        </button>
        <button v-if="filters.serverId !== 'all'" class="secondary-button" type="button" @click="openLogQueryPage(filters.serverId)">
          <Search :size="18" />
          <span>查日志 {{ selectedServerName }}</span>
        </button>
        <button class="icon-button ghost" type="button" title="设置" @click="openSettingsPage">
          <Settings :size="18" />
        </button>
        <button class="icon-button ghost" type="button" title="退出登录" @click="logout">
          <LogOut :size="18" />
        </button>
      </div>
    </section>

    <section class="server-tabs" aria-label="服务器">
      <button
        v-for="server in serverTabs"
        :key="server.serverId"
        type="button"
        :class="{ active: filters.serverId === server.serverId }"
        @click="selectServer(server.serverId)"
      >
        <Layers :size="16" />
        <span>{{ server.serverName }}</span>
        <small v-if="server.serverId !== 'all'">{{ formatNumber(sourceStat(server.serverId, 'totalCount')) }}</small>
      </button>
    </section>

    <section class="filters" aria-label="筛选">
      <label>
        <span>开始</span>
        <input v-model="filters.from" type="date" @change="loadAll" />
      </label>
      <label>
        <span>结束</span>
        <input v-model="filters.to" type="date" @change="loadAll" />
      </label>
      <label class="search-field">
        <span>玩家</span>
        <div class="search-box">
          <Search :size="17" />
          <input v-model.trim="filters.player" type="search" placeholder="名称" @keyup.enter="loadAll" />
        </div>
      </label>
      <button class="secondary-button" type="button" @click="loadAll">
        查询
      </button>
    </section>

    <p v-if="error" class="notice error">{{ error }}</p>
    <p v-if="importMessage" class="notice success">{{ importMessage }}</p>

    <section class="metric-grid">
      <article class="metric-card">
        <div class="metric-icon red"><Hammer :size="20" /></div>
        <p>破坏方块</p>
        <strong>{{ formatNumber(overview.brokenCount) }}</strong>
      </article>
      <article class="metric-card">
        <div class="metric-icon green"><Blocks :size="20" /></div>
        <p>放置方块</p>
        <strong>{{ formatNumber(overview.placedCount) }}</strong>
      </article>
      <article class="metric-card">
        <div class="metric-icon blue"><CalendarDays :size="20" /></div>
        <p>玩家数量</p>
        <strong>{{ formatNumber(overview.playerCount) }}</strong>
      </article>
      <article class="metric-card">
        <div class="metric-icon amber"><Database :size="20" /></div>
        <p>导入文件</p>
        <strong>{{ formatNumber(overview.importedFileCount) }}</strong>
        <small>{{ formatDateTime(overview.lastImportedAt) }}</small>
      </article>
    </section>

    <section class="content-grid">
      <article class="panel wide">
        <div class="panel-title">
          <h2>玩家排行</h2>
          <span>{{ selectedServerName }} · {{ formatNumber(overview.totalCount) }} 次方块行为</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>玩家</th>
                <th>破坏</th>
                <th>放置</th>
                <th>合计</th>
                <th>首次记录</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="player in players" :key="player.playerName">
                <td>{{ player.playerName }}</td>
                <td>{{ formatNumber(player.brokenCount) }}</td>
                <td>{{ formatNumber(player.placedCount) }}</td>
                <td>{{ formatNumber(player.totalCount) }}</td>
                <td>{{ formatDateTime(player.firstSeenAt) }}</td>
              </tr>
              <tr v-if="!players.length">
                <td colspan="5" class="empty">暂无数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

      <article class="panel">
        <div class="panel-title">
          <h2>活跃玩家</h2>
          <span>Top {{ leadingPlayers.length }}</span>
        </div>
        <div class="rank-list">
          <div v-for="player in leadingPlayers" :key="player.playerName" class="rank-row">
            <span>{{ player.playerName }}</span>
            <div class="rank-bar">
              <i :style="{ width: `${Math.max((player.totalCount / Math.max(overview.totalCount, 1)) * 100, 3)}%` }"></i>
            </div>
            <strong>{{ formatNumber(player.totalCount) }}</strong>
          </div>
          <p v-if="!leadingPlayers.length" class="empty">暂无数据</p>
        </div>
      </article>

      <article class="panel wide">
        <div class="panel-title">
          <h2>每日走势</h2>
          <span>{{ daily.length }} 天</span>
        </div>
        <div class="trend">
          <div v-for="day in daily" :key="day.statDate" class="trend-row">
            <time>{{ day.statDate.slice(5) }}</time>
            <div class="trend-track">
              <i class="broken" :style="{ width: `${(day.brokenCount / maxDailyTotal) * 100}%` }"></i>
              <i class="placed" :style="{ width: `${(day.placedCount / maxDailyTotal) * 100}%` }"></i>
            </div>
            <strong>{{ formatNumber(day.totalCount) }}</strong>
          </div>
          <p v-if="!daily.length" class="empty">暂无数据</p>
        </div>
      </article>

      <article class="panel">
        <div class="panel-title">
          <h2>最近导入</h2>
          <span>{{ imports.length }} 个文件</span>
        </div>
        <div class="import-list">
          <div v-for="file in imports" :key="`${file.serverId}:${file.remotePath}`" class="import-row">
            <div>
              <strong>{{ file.fileName }}</strong>
              <small>{{ file.serverName }} · {{ file.logDate || '-' }} · {{ fileSize(file.fileSize) }}</small>
            </div>
            <span>{{ formatNumber(file.rowCount - file.ignoredCount) }}</span>
          </div>
          <p v-if="!imports.length" class="empty">暂无数据</p>
        </div>
      </article>
    </section>
    </template>
  </main>
</template>
