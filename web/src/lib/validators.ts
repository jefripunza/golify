// Zod validation schemas — shared by Login + Onboarding.
import { z } from 'zod'
import { DOMAIN_SUFFIXES } from './domainSuffixes'

export const emailSchema = z
  .string()
  .trim()
  .min(1, 'Email is required')
  .email('Invalid email format')

// Strong password policy:
// - min 8 chars
// - at least one lowercase, one uppercase, one digit, one symbol
export const strongPasswordSchema = z
  .string()
  .min(8, 'Password must be at least 8 characters')
  .regex(/[a-z]/, 'Must include a lowercase letter (a-z)')
  .regex(/[A-Z]/, 'Must include an uppercase letter (A-Z)')
  .regex(/[0-9]/, 'Must include a number (0-9)')
  .regex(/[^a-zA-Z0-9]/, 'Must include a symbol (e.g. !@#$%)')

export const loginSchema = z.object({
  email: emailSchema,
  password: z.string().min(1, 'Password is required'),
})

export const onboardSchema = z
  .object({
    email: emailSchema,
    password: strongPasswordSchema,
    confirm: z.string(),
  })
  .refine((d) => d.password === d.confirm, {
    message: 'Passwords do not match',
    path: ['confirm'],
  })

export type LoginValues = z.infer<typeof loginSchema>
export type OnboardValues = z.infer<typeof onboardSchema>

// ── Domain validation ─────────────────────────────────────────────────────
// Accepts ROOT DOMAIN (2 labels, e.g. "example.com") AND subdomains
// (3+ labels, e.g. "sub.example.com"). Auto-strips http:// / https://
// scheme, paths, query strings, and a leading "www." label
// (www.example.com ≡ example.com). Final suffix must be a real domain
// suffix from the IANA/PSL-backed DOMAIN_SUFFIXES set.
export const domainSchema = z
  .string()
  .trim()
  .min(1, 'Domain is required')
  .transform((raw) => {
    // strip scheme if present
    let d = raw.trim()
    if (d.includes('://')) d = d.slice(d.indexOf('://') + 3)
    // strip trailing path/query/fragment
    d = d.replace(/[/?#].*$/, '')
    d = d.replace(/\.+$/, '').toLowerCase()
    // www.example.com ≡ example.com — drop the www. label
    d = d.replace(/^www\./, '')
    return d
  })
  .refine((d) => d.length > 0, { message: 'Domain is required' })
  // one or more labels: label.label or sub.label.label
  .refine((d) => /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$/.test(d), {
    message: 'Invalid domain format (e.g. example.com or sub.example.com).',
  })
  .refine((d) => {
    const labels: string[] = d.split('.')
    return labels.every((l: string) => l && l[0] !== '-' && l[l.length - 1] !== '-')
  }, { message: 'Domain labels must not start/end with "-"' })
  .refine((d) => {
    // suffix must be a real TLD / public suffix (e.g. .com, .id, .co.id)
    // AND at least one label must precede it — a bare suffix (co.id, com)
    // is NOT a valid domain. Tries the longest suffix first.
    const labels: string[] = d.split('.')
    for (let i = 0; i < labels.length; i++) {
      const suffix = labels.slice(i).join('.')
      if (DOMAIN_SUFFIXES.has(suffix)) return i > 0
    }
    return false
  }, { message: 'Invalid domain suffix (must end with a valid TLD like .com, .id, .online, .co.id)' })

export type DomainValues = z.infer<typeof domainSchema>

