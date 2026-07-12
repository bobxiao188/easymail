export const getCookie = (name: string): string | null => {
  if (document.cookie.length > 0) {
    const start = document.cookie.indexOf(name + '=')
    if (start !== -1) {
      const startPos = start + name.length + 1
      let end = document.cookie.indexOf(';', startPos)
      if (end === -1) end = document.cookie.length
      return decodeURIComponent(document.cookie.substring(startPos, end))
    }
  }
  return null
}

export const setCookie = (name: string, value: string, days: number): void => {
  let expires = ''
  if (days) {
    const date = new Date()
    date.setTime(date.getTime() + days * 24 * 60 * 60 * 1000)
    expires = '; expires=' + date.toUTCString()
  }
  document.cookie = name + '=' + encodeURIComponent(value) + expires + '; path=/'
}

export const deleteCookie = (name: string): void => {
  document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/'
}
