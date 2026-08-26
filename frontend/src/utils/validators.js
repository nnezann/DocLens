const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function isValidEmailFormat(email) {
  return EMAIL_REGEX.test(email.trim())
}

// --- Existence check -------------------------------------------------
// This is a MOCK. There is no backend in this project, so we can't really
// ask a mail server "does this mailbox exist". In a real app, swap the
// body of this function for a call to your backend, which would in turn
// use a service like ZeroBounce, AbstractAPI, or Kickbox, or simply check
// against your own users table for login/forgot-password flows.
const MOCK_KNOWN_EMAILS = new Set(['demo@work.com', 'mark.ferdinand@work.com'])

export async function checkEmailExists(email) {
  await new Promise((resolve) => setTimeout(resolve, 500)) // simulate network latency
  return MOCK_KNOWN_EMAILS.has(email.trim().toLowerCase())
}

// --- Password strength -------------------------------------------------
export const PASSWORD_RULES = [
  { id: 'length', label: 'At least 8 characters', test: (pw) => pw.length >= 8 },
  { id: 'uppercase', label: 'One uppercase letter (A-Z)', test: (pw) => /[A-Z]/.test(pw) },
  { id: 'lowercase', label: 'One lowercase letter (a-z)', test: (pw) => /[a-z]/.test(pw) },
  { id: 'number', label: 'One number (0-9)', test: (pw) => /\d/.test(pw) },
  { id: 'special', label: 'One special character (!@#$...)', test: (pw) => /[^A-Za-z0-9]/.test(pw) },
]

export function getPasswordStrength(password) {
  const results = PASSWORD_RULES.map((rule) => ({ ...rule, met: rule.test(password) }))
  const metCount = results.filter((r) => r.met).length
  return { results, metCount, isStrong: metCount === PASSWORD_RULES.length }
}