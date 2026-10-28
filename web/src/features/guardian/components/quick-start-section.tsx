/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { cn } from '@/lib/utils'

import { GUARDIAN_INSTALL_COMMAND } from '../constants'
import { GuardianSection, SectionHeading } from './section-heading'

function CodeBlock(props: { label: string; code: string; wrap?: boolean }) {
  const { t } = useTranslation()
  return (
    <div className='overflow-hidden rounded-xl bg-[var(--g-code-bg)] text-[var(--g-code-ink)]'>
      <div className='flex items-center justify-between border-b border-white/10 px-4 py-2'>
        <span className='g-mono text-xs text-[var(--g-code-soft)]'>
          {props.label}
        </span>
        <CopyButton
          value={props.code}
          variant='ghost'
          size='icon'
          className='size-7 text-[var(--g-code-soft)] hover:bg-white/10 hover:text-[var(--g-code-ink)]'
          tooltip={t('Copy')}
        />
      </div>
      <pre
        className={cn(
          'g-mono overflow-x-auto px-4 py-4 text-[0.8125rem] leading-6',
          props.wrap && 'break-words whitespace-pre-wrap'
        )}
      >
        <code>{props.code}</code>
      </pre>
    </div>
  )
}

export function QuickStartSection() {
  const { t } = useTranslation()
  const firstPrompt = `$claude-guardian\n${t(
    'First time subscribing to Claude, on a Mac. Compare local and VM first, then decide what to check.'
  )}`
  const steps = [
    {
      title: t('Install the Skill'),
      body: t(
        'Copy the claude-guardian folder into your Codex skills directory. The command stops instead of overwriting an existing Skill.'
      ),
      code: { label: 'bash', value: GUARDIAN_INSTALL_COMMAND, wrap: false },
    },
    {
      title: t('Start a conversation'),
      body: t(
        'The agent asks your stage, system and goal first, and skips what you have already answered.'
      ),
      code: { label: 'codex', value: firstPrompt, wrap: true },
    },
    {
      title: t('Pick a route, then check'),
      body: t(
        'Say what you want to do, choose local, VM or VPS, confirm the check scope, then approve plans by tier. Only comparing options? It stops at the explanation and never touches your device.'
      ),
    },
  ]

  return (
    <GuardianSection id='quick-start' tone='sunk'>
      <div className='grid gap-14 lg:grid-cols-[0.8fr_1.2fr]'>
        <SectionHeading
          id='quick-start'
          eyebrow={t('Quick start')}
          title={t('Free, open source, ready in minutes.')}
          lede={t(
            'Requires Codex with Skills support. Scripts need Python 3.10 or later; the references can be read without running anything.'
          )}
        />
        <ol className='space-y-10'>
          {steps.map((step, index) => (
            <li key={step.title} className='grid gap-4 sm:grid-cols-[3rem_1fr]'>
              <span className='g-numeral text-3xl'>{index + 1}</span>
              <div className='min-w-0 space-y-4'>
                <h3 className='g-heading text-xl'>{step.title}</h3>
                <p className='text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
                  {step.body}
                </p>
                {step.code && (
                  <CodeBlock
                    label={step.code.label}
                    code={step.code.value}
                    wrap={step.code.wrap}
                  />
                )}
              </div>
            </li>
          ))}
        </ol>
      </div>
    </GuardianSection>
  )
}
