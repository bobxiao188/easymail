/**
 * Folder type definitions
 * Fully matches backend easymail/internal/domain/mailbox/folder.go definitions
 */

// Folder types (corresponds to backend FolderKind enum)
export const FolderKind = {
  Unknown: 0,
  Inbox: 1,
  Sent: 2,
  Draft: 3,
  Trash: 4,
  Spam: 5,
  Quarantine: 6,
  UserCustomMin: 100,
} as const;

// System folder kinds (excluding Unknown)
export type SystemFolderKind = typeof FolderKind.Inbox | typeof FolderKind.Sent | typeof FolderKind.Draft | typeof FolderKind.Trash | typeof FolderKind.Spam | typeof FolderKind.Quarantine;

// All folder kinds including Unknown and custom folders
export type FolderKind = SystemFolderKind | number;

// System folder type range
export const IS_SYSTEM_FOLDER_KIND = (kind: FolderKind): boolean => {
  return kind >= 1 && kind < FolderKind.UserCustomMin
}

// Folder kind to route slug mapping
export const FOLDER_ROUTE_MAP: Record<number, string> = {
  [FolderKind.Inbox]: 'inbox',
  [FolderKind.Sent]: 'sent',
  [FolderKind.Draft]: 'drafts',
  [FolderKind.Trash]: 'trash',
  [FolderKind.Spam]: 'spam',
  [FolderKind.Quarantine]: 'quarantine',
}