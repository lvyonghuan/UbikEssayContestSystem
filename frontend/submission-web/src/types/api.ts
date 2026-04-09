export interface ApiResponse<T = unknown> {
  code: number
  msg: T
}

export type JsonObject = Record<string, unknown>

export interface AuthorLoginPayload {
  authorName?: string
  authorEmail?: string
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

export interface SubmissionPayload {
  workTitle: string
  trackID: number
  workInfos?: JsonObject
}

export interface SubmissionUpdatePayload extends SubmissionPayload {
  workID: number
}

export type ContestStatus = '未开始' | '进行中' | '已结束'
