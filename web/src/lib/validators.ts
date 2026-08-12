// Zod validation schemas — shared by Login + Onboarding.
import { z } from 'zod'

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
// Accepts ROOT DOMAIN ONLY (2 labels, e.g. "example.com"). Subdomains
// (3+ labels, e.g. "sub.example.com") are REJECTED. Auto-strips
// http:// / https:// scheme, paths, query strings.
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
    return d
  })
  .refine((d) => d.length > 0, { message: 'Domain is required' })
  // exactly two labels: label.label (root domain, NO subdomain)
  .refine((d) => /^[a-z0-9]([a-z0-9-]*[a-z0-9])?\.[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(d), {
    message: 'Only root domains are allowed (e.g. example.com). Subdomains are not permitted.',
  })
  .refine((d) => {
    const labels: string[] = d.split('.')
    return labels.length === 2 && labels.every((l: string) => l && l[0] !== '-' && l[l.length - 1] !== '-')
  }, { message: 'Domain labels must not start/end with "-"' })

export type DomainValues = z.infer<typeof domainSchema>
