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
  scriptKey?: string
  scriptName: string
  description?: string
  scriptDescription?: string
  interpreter?: string
  isEnabled?: boolean
  activeVersionID?: number
  meta?: JsonObject
  extensionData?: JsonObject
  createdAt?: string
  updatedAt?: string
}

export interface ScriptVersion {
  versionID?: number
  scriptID?: number
  versionNum?: number
  versionName?: string
  fileName?: string
  relativePath?: string
  fileURL?: string
  checksum?: string
  fileHash?: string
  createdBy?: number
  isActive?: boolean
  extensionData?: JsonObject
  createdAt?: string
}

export interface ScriptDefinitionStatusPayload {
  isEnabled: boolean
}

export interface ScriptFlow {
  flowID?: number
  flowKey?: string
  flowName: string
  description?: string
  flowDescription?: string
  isEnabled?: boolean
  meta?: JsonObject
  extensionData?: JsonObject
  createdAt?: string
  updatedAt?: string
}

export type FlowFailureStrategy = 'fail_close' | 'fail_open' | 'retry'

export interface FlowStep {
  stepID?: number
  flowID?: number
  stepOrder: number
  stepName?: string
  scriptID?: number
  scriptVersionID?: number
  isEnabled?: boolean
  failureStrategy?: FlowFailureStrategy | string
  inputTemplate?: JsonObject
  timeoutMs?: number
  stepConfig?: JsonObject
  extensionData?: JsonObject
}

export type FlowMountScope = 'submission' | 'system' | 'judge'
export type FlowMountTargetType = 'global' | 'contest' | 'track'

export interface FlowMount {
  mountID?: number
  flowID?: number
  scope?: FlowMountScope | string
  targetType?: FlowMountTargetType | string
  targetID?: number
  eventKey?: string
  isEnabled?: boolean
  containerType?: FlowMountTargetType | string
  containerID?: number
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
  workStatus?: string
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
  workStatus?: string
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

export interface JudgeAccountInput {
  judgeName: string
  password: string
}

export interface ReviewEventInput {
  trackID: number
  eventName: string
  // workStatus is a filter condition for selecting works in the event scope.
  workStatus?: string
  startTime?: string
  endTime?: string
}

export interface ReviewEvent extends ReviewEventInput {
  eventID: number
  judgeIDs?: number[]
}

export interface JudgeProgressStat {
  judgeID: number
  judgeName: string
  assignedCount: number
  submittedCount: number
  completionRate: number
}

export interface ReviewEventProgress {
  eventID: number
  eventName: string
  trackID: number
  assignedJudgeIDs: number[]
  totalWorks: number
  completedWorks: number
  judgeProgress: JudgeProgressStat[]
}

export interface WorkEventReview {
  eventID: number
  eventName: string
  assignedJudges: number
  submittedReviews: number
  completed: boolean
}

export interface WorkReviewStatus {
  workID: number
  events: WorkEventReview[]
  summary: Record<string, number>
  meta?: Record<string, string>
}

export interface ReviewResult {
  resultID: number
  workID: number
  reviewEventID: number
  reviews: JsonObject
}

export interface TrackRankItem {
  workID: number
  workTitle: string
  authorID: number
  authorName: string
  finalScore: number
  reviewCount: number
}

export interface DashboardOverview {
  trackSubmissionCount: Record<string, number>
  participatingAuthors: number
  completedJudgeTasks: number
  totalTrackJudges: number
  completedReviewedWorks: number
}

export interface ContestTrackStatusStat {
  trackID: number
  trackName: string
  totalWorks: number
  statusCounts: Record<string, number>
  totalAuthors: number
  distinctStates: string[]
}

export interface ContestDailySubmissionStat {
  date: string
  count: number
}

export interface ContestReviewRegenerateResult {
  generated: number
}

export interface JudgeDeadlinePayload {
  deadlineAt: string
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
