import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface Mail {
  id: number
  sender: string
  senderEmail: string
  avatarInitials: string
  subject: string
  preview: string
  body: string
  time: string
  read: boolean
  starred: boolean
  important: boolean
  hasAttachments: boolean
  folder: 'inbox' | 'sent' | 'drafts' | 'deleted' | 'archive' | 'favorites'
}

interface Contact {
  id: number
  name: string
  email: string
  company: string
  jobTitle: string
  phone: string
  avatar: string
}

const senders = [
  { name: 'Alice Johnson', email: 'alice@example.com' },
  { name: 'Bob Smith', email: 'bob@example.com' },
  { name: 'Carol Davis', email: 'carol@example.com' },
  { name: 'David Lee', email: 'david@example.com' },
  { name: 'Eve Martin', email: 'eve@example.com' },
  { name: 'Frank Wilson', email: 'frank@example.com' },
  { name: 'Grace Chen', email: 'grace@example.com' },
  { name: 'Henry Brown', email: 'henry@example.com' },
  { name: 'Ivy Taylor', email: 'ivy@example.com' },
  { name: 'Jack White', email: 'jack@example.com' },
]

const subjects = [
  'Q4 Budget Review',
  'Meeting Tomorrow',
  'Project Update: Dashboard',
  'Follow-up on Proposal',
  'Team Offsite Planning',
  'Invoice Attached',
  'Security Alert',
  'New Feature Launch',
  'Lunch Tomorrow?',
  'Weekly Standup Notes',
  'Vacation Request',
  'Onboarding Documents',
  'Design Review',
  'Client Feedback',
  'System Maintenance',
  'Happy Birthday!',
  'Performance Review',
  'Data Export Ready',
  'Conference Registration',
  'Partnership Opportunity',
]

function generatePreview(): string {
  const pre = [
    'Please find attached the...',
    'Let me know if you have any...',
    'I wanted to follow up on...',
    'Thanks for the update...',
    'Looking forward to your...',
  ]
  return pre[Math.floor(Math.random() * pre.length)]
}

function generateBody(subject: string): string {
  return `<p>Hi,</p><p>This email is regarding <strong>${subject}</strong>.</p><p>Please review the details and let me know your thoughts. We can schedule a call if needed.</p><p>Best regards,<br/>[Sender]</p>`
}

function randomTime(): string {
  const now = Date.now()
  const offset = Math.floor(Math.random() * 7 * 24 * 60 * 60 * 1000)
  const date = new Date(now - offset)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  return `${hours}:${minutes}`
}

function createMockEmails(count: number): Mail[] {
  const mails: Mail[] = []
  const folders: Mail['folder'][] = ['inbox', 'sent', 'drafts', 'deleted', 'archive', 'favorites']
  
  for (let i = 0; i < count; i++) {
    const sender = senders[Math.floor(Math.random() * senders.length)]
    const subject = subjects[Math.floor(Math.random() * subjects.length)]
    const folder = folders[Math.floor(Math.random() * folders.length)]
    
    mails.push({
      id: i + 1,
      sender: sender.name,
      senderEmail: sender.email,
      avatarInitials: sender.name.split(' ').map(n => n[0]).join('').toUpperCase(),
      subject,
      preview: generatePreview(),
      body: generateBody(subject),
      time: randomTime(),
      read: Math.random() > 0.4,
      starred: Math.random() > 0.8,
      important: Math.random() > 0.9,
      hasAttachments: Math.random() > 0.7,
      folder: folder,
    })
  }
  return mails
}

export const useMailStore = defineStore('mail', () => {
  const mails = ref<Mail[]>(createMockEmails(50))
  const selectedMailId = ref<number | null>(null)
  const activeFolder = ref<string>('inbox')
  const density = ref<'compact' | 'cozy' | 'roomy'>('cozy')
  const panePosition = ref<'right' | 'bottom' | 'off'>('right')
  const searchQuery = ref('')
  const darkMode = ref(false)
  
  // Contacts
  // Contacts
  const contacts = ref<Contact[]>([
    { id: 1, name: 'Alice Johnson', email: 'alice@example.com', company: 'Acme Corp', jobTitle: 'Product Manager', phone: '+1 555-0101', avatar: 'https://i.pravatar.cc/150?img=1' },
    { id: 2, name: 'Bob Smith', email: 'bob@example.com', company: 'Tech Inc', jobTitle: 'Engineer', phone: '+1 555-0102', avatar: 'https://i.pravatar.cc/150?img=2' },
    { id: 3, name: 'Carol Davis', email: 'carol@example.com', company: 'Design Co', jobTitle: 'Designer', phone: '+1 555-0103', avatar: 'https://i.pravatar.cc/150?img=3' },
    { id: 4, name: 'David Lee', email: 'david@example.com', company: 'Sales Ltd', jobTitle: 'Sales Director', phone: '+1 555-0104', avatar: 'https://i.pravatar.cc/150?img=4' },
    { id: 5, name: 'Eve Martin', email: 'eve@example.com', company: 'Marketing Pro', jobTitle: 'Marketing Manager', phone: '+1 555-0105', avatar: 'https://i.pravatar.cc/150?img=5' },
    { id: 6, name: 'Frank Wilson', email: 'frank@example.com', company: 'Finance Group', jobTitle: 'CFO', phone: '+1 555-0106', avatar: 'https://i.pravatar.cc/150?img=6' },
    { id: 7, name: 'Grace Chen', email: 'grace@example.com', company: 'HR Solutions', jobTitle: 'HR Director', phone: '+1 555-0107', avatar: 'https://i.pravatar.cc/150?img=7' },
    { id: 8, name: 'Henry Brown', email: 'henry@example.com', company: 'Legal Associates', jobTitle: 'Attorney', phone: '+1 555-0108', avatar: 'https://i.pravatar.cc/150?img=8' },
    { id: 9, name: 'Ivy Taylor', email: 'ivy@example.com', company: 'Operations Plus', jobTitle: 'Operations Manager', phone: '+1 555-0109', avatar: 'https://i.pravatar.cc/150?img=9' },
    { id: 10, name: 'Jack White', email: 'jack@example.com', company: 'Strategy Partners', jobTitle: 'Consultant', phone: '+1 555-0110', avatar: 'https://i.pravatar.cc/150?img=10' },
  ])
  
  const filteredMails = computed(() => {
    let list = mails.value.filter(m => m.folder === activeFolder.value || (activeFolder.value === 'favorites' ? true : false))
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      list = list.filter(m => m.subject.toLowerCase().includes(q) || m.sender.toLowerCase().includes(q) || m.preview.toLowerCase().includes(q))
    }
    return list
  })

  const selectedMail = computed(() => mails.value.find(m => m.id === selectedMailId.value) || null)

  function selectMail(id: number) {
    selectedMailId.value = id
    const mail = mails.value.find(m => m.id === id)
    if (mail) mail.read = true
  }

  function setFolder(folder: string) {
    activeFolder.value = folder
    selectedMailId.value = null
  }

  function toggleStar(id: number) {
    const mail = mails.value.find(m => m.id === id)
    if (mail) mail.starred = !mail.starred
  }

  function toggleDark() {
    darkMode.value = !darkMode.value
  }

  return { 
    mails, 
    selectedMailId, 
    activeFolder, 
    density, 
    panePosition, 
    searchQuery, 
    darkMode, 
    filteredMails, 
    selectedMail, 
    selectMail, 
    setFolder, 
    toggleStar,
    toggleDark,
    contacts
  }
})