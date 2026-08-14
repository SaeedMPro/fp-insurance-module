import { useEffect, useState } from 'react'

/**
 * Theme preference: explicit light/dark, or "system" (follow the OS).
 * The choice lives in localStorage("theme"); "system" is stored as absence so a
 * fresh visitor gets the OS theme with zero flash (see the inline boot script
 * in index.html, which applies the class before React mounts).
 */
export type ThemePref = 'light' | 'dark' | 'system'

const KEY = 'theme'

function systemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function apply(pref: ThemePref) {
  const dark = pref === 'dark' || (pref === 'system' && systemPrefersDark())
  document.documentElement.classList.toggle('dark', dark)
}

export function useTheme() {
  const [pref, setPref] = useState<ThemePref>(
    () => (localStorage.getItem(KEY) as ThemePref | null) ?? 'system',
  )

  useEffect(() => {
    if (pref === 'system') {
      localStorage.removeItem(KEY)
    } else {
      localStorage.setItem(KEY, pref)
    }
    apply(pref)

    if (pref !== 'system') return
    // In "system" mode, live-follow OS theme changes.
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => apply('system')
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [pref])

  return { pref, setPref, isDark: pref === 'dark' || (pref === 'system' && systemPrefersDark()) }
}
