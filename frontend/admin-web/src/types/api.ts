export interface ApiResponse<T = unknown> {
  code: number
  msg: T
}

export type JsonObject = Record<string, unknown>

export interface AdminLoginPayload {
  adminName: string
  password: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface Contest {
  contestID?: number
  contestName: string
  contestIntroduction?: string
  contestStartDate: string
  contestEndDate: string
}

export interface Track {
  trackID?: number
  contestID?: number
  trackName: string
  trackDescription?: string
  trackSettings?: Record<string, unknown>
}

export interface GlobalConfig {
  siteName: string
  emailAddress: string
  emailSmtpServer: string
  emailSmtpPort: string
}

export interface RolePermission {
  roleID: number
  roleName: string
  description: string
  permissions: string[]
}

export interface AdminProfile {
  adminID: number
  adminName: string
  adminEmail?: string
  roleNames: string[]
}

export interface SubAdminInfo {
  adminID: number
  adminName: string
  adminEmail: string
  isActive: boolean
  permissionNames: string[]
}

export interface CreateSubAdminRequest {
  adminName: string
  adminEmail: string
  permissionNames?: string[]
}

export interface SubAdminCreateResult {
  adminID: number
  adminName: string
  adminEmail: string
  tempPassword?: string
  emailSent?: boolean
  emailError?: string
}

export interface UpdateSubAdminPermissionsRequest {
  permissionNames: string[]
}

export interface DashboardSummary {
  totalContests: number
  totalTracks: number
  totalWorks: number
  totalReviewEvents: number
}

export interface ScriptDefinition {
  scriptID?: number
  scriptName: string
  scriptDescription?: string
  isEnabled?: boolean
  activeVersionID?: number
  extensionData?: JsonObject
  createdAt?: string
  updatedAt?: string
}

export interface ScriptVersion {
  versionID?: number
  scriptID?: number
  versionName?: string
  fileName?: string
  fileURL?: string
  fileHash?: string
  isActive?: boolean
  extensionData?: JsonObject
  createdAt?: string
}

export interface ScriptDefinitionStatusPayload {
  isEnabled: boolean
}

export interface ScriptFlow {
  flowID?: number
  flowName: string
  flowDescription?: string
  isEnabled?: boolean
  extensionData?: JsonObject
  createdAt?: string
  updatedAt?: string
}

export interface FlowStep {
  stepID?: number
  flowID?: number
  stepOrder: number
  stepName?: string
  scriptID?: number
  stepConfig?: JsonObject
  extensionData?: JsonObject
}

export interface FlowMount {
  mountID?: number
  flowID?: number
  containerType: string
  containerID: number
  mountConfig?: JsonObject
  extensionData?: JsonObject
  createdAt?: string
}

export interface ScriptFlowStatusPayload {
  isEnabled: boolean
}

export interface Work {
  workID?: number
  authorID?: number
  authorName?: string
  trackID?: number
  trackName?: string
  workTitle?: string
  workInfos?: JsonObject
}

export interface Author {
  authorID?: number
  authorName: string
  authorEmail?: string
  penName?: string
  password?: string
  authorInfos?: JsonObject
}

export interface AuthorQueryParams {
  authorName?: string
  offset?: number
  limit?: number
}

export interface WorkQueryParams {
  trackID?: number
  workTitle?: string
  authorName?: string
  offset?: number
  limit?: number
}

export interface JudgeProfile {
  judgeID: number
  judgeName: string
  judgeEmail?: string
  isActive?: boolean
}

export interface ContestJudgeBinding {
  contestID: number
  judgeID: number
  role?: string
}

export interface DashboardTrendPoint {
  date: string
  count: number
}

export interface DashboardDistributionPoint {
  name: string
  value: number
}

export interface DashboardMetrics {
  scriptTotal: number
  scriptEnabledTotal: number
  flowTotal: number
  flowEnabledTotal: number
  flowMountTotal: number
  workTotal: number
  trendData: DashboardTrendPoint[]
  trackDistribution: DashboardDistributionPoint[]
  workStatusData: DashboardDistributionPoint[]
}
