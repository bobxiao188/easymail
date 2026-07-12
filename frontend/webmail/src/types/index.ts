// Import FolderKind type
import type { FolderKind } from '../utils/folder'

// User type
export interface User {
  id: string
  email: string
  name: string
  avatar?: string
  phone?: string
  company?: string
  jobTitle?: string
  storageUsed: number
  storageLimit: number
}

// User settings type
export interface UserSettings {
  displayName: string
  signature: string
  language: string
  theme: string
  pageSize: number
  readingPanePosition: string
  autoReplyEnabled: boolean
  autoReplySubject: string
  autoReplyBody: string
  phone: string
  company: string
  jobTitle: string
  notificationSound: boolean
  desktopNotification: boolean
  includeOriginalOnReply: boolean
  forwardingEnabled: boolean
  forwardingAddress: string
  saveSent: boolean
}

// Email type
export interface Email {
  id: number
  subject: string
  from: EmailAddress
  to: EmailAddress[]
  cc: EmailAddress[]
  bcc: EmailAddress[]
  body: string
  bodyHtml?: string
  attachments: Attachment[]
  mailTime: string  // Field name returned by backend
  createdAt?: string  // Compatible with old code
  updatedAt?: string
  isRead: boolean
  isStarred: boolean
  folderId: number
  labels: LabelItem[]
  headers?: Record<string, string>
}

// Email list item type (used for list display)
export interface EmailListItem {
  id: number
  subject: string
  from: EmailAddress
  to?: Recipient[]  // Recipient list
  cc?: Recipient[]  // CC list
  snippet: string
  mailTime: string  // Field name returned by backend
  createdAt?: string  // Compatible with old code
  isRead: boolean
  isStarred: boolean
  hasAttachments: boolean
  folderId: number
  labels: LabelItem[]
}

// Label item from API (id, name, color)
export interface LabelItem {
  id: number
  name: string
  color: string
}

// Sender/recipient address
export interface EmailAddress {
  name: string
  email: string
}

// Recipient type (used for sending emails)
export interface Recipient {
  name: string
  email: string
}

// Unified email composition interface (used for saving drafts, updating drafts, sending)
export interface ComposeMessage {
  subject: string
  text?: string
  html?: string
  from?: Recipient
  to: Recipient[]
  cc?: Recipient[]
  bcc?: Recipient[]
  attachments?: ComposeAttachment[]
  saveSent?: boolean
  signature?: string
  draftId?: number // Draft ID, used to delete draft after sending
  folderId?: number // Folder ID, used to specify save location (e.g., drafts folder)
}

// Attachment upload data
export interface ComposeAttachment {
  name: string
  size: number
  base64: string
}

// Draft detail (EditDraft response)
export interface DraftDetail {
  id: number
  subject: string
  text: string
  html?: string
  to: Recipient[]
  cc: Recipient[]
  bcc: Recipient[]
  from?: Recipient
  mailTime: string
  folderId: number
  attachments?: ComposeAttachment[]
}

// Attachment type
export interface Attachment {
  id: number
  index: number
  name: string
  size: number
  mimeType: string
  url: string
  content_type?: string
}

// Pagination request parameters
export interface PaginationParams {
  page: number
  pageSize: number
  keyword?: string
  folderId?: number
  labelId?: number // Filter by label ID (optional, 0 or undefined means no filter)
  startDate?: Date
  endDate?: Date
  isRead?: boolean
  isStarred?: boolean
}

// API response type
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

// Contact group type (consistent with backend ContactGroupDTO)
export interface ContactGroup {
  id: string
  groupName: string
  isDefault: boolean
  createTime: string
  contactCount?: number // Field additionally added by frontend
}

// Contact type (consistent with backend ContactDTO)
export interface Contact {
  id: string
  contactName: string
  contactEmail: string
  contactPhone?: string
  contactAddress?: string
  contactCity?: string
  contactState?: string
  contactZip?: string
  contactCountry?: string
  contactGroupId?: string | null
}

// Pagination response
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

// Input when creating/updating contact (consistent with backend ContactInput, excluding id)
export type ContactInput = Omit<Contact, 'id'>

// Email statistics
export interface MailStats {
  inboxCount: number
  unreadCount: number
  sentCount: number
  draftCount: number
  trashCount: number
  spamCount: number
  storageUsed: number
  storageLimit: number
}

// Search parameters
export interface SearchParams {
  keyword: string
  folderId?: number
  startDate?: Date
  endDate?: Date
}


// Email labels
export interface Label {
  id: number
  name: string
  color: string
  isBuiltin: boolean
  emailCount?: number
}

// Folder type
export interface Folder {
  id: number
  name: string
  imapName: string
  kind: FolderKind
  unreadCount?: number
  totalCount?: number
}
