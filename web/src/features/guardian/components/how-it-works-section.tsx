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

import { AnimateInView } from '@/components/animate-in-view'
import { cn } from '@/lib/utils'

import { GuardianSection, SectionHeading } from './section-heading'

type FlowNode = {
  label: string
  title: string
  body: string
  emphasis?: boolean
}

function FlowCard(props: { node: FlowNode }) {
  return (
    <div
      className={cn(
        'g-rule rounded-xl border px-5 py-5',
        props.node.emphasis
          ? 'border-[var(--g-ink)] bg-[var(--g-paper-raised)]'
          : 'bg-[var(--g-paper)]'
      )}
    >
      <p className='g-mono text-xs text-[var(--g-ink-soft)]'>
        {props.node.label}
      </p>
      <p className='g-heading mt-2 text-lg'>{props.node.title}</p>
      <p className='mt-2 text-sm leading-6 text-[var(--g-ink-soft)]'>
        {props.node.body}
      </p>
    </div>
  )
}

function FlowConnector() {
  return (
    <div aria-hidden='true' className='flex justify-center'>
      <span className='g-connector-v block h-8 w-px lg:hidden' />
      <span className='g-connector hidden h-px w-full lg:block' />
    </div>
  )
}

export function HowItWorksSection() {
  const { t } = useTranslation()
  const inputs: FlowNode[] = [
    {
      label: t('You'),
      title: t('Needs and environment'),
      body: t('Your stage, system, network resources, budget and goal.'),
    },
    {
      label: 'Skill',
      title: t('Experience, references, rules'),
      body: t(
        'Field notes from the author and community, a source library and authorization tiers.'
      ),
    },
  ]
  const agent: FlowNode = {
    label: 'Agent',
    title: t('Understands, judges, explains'),
    body: t(
      'Picks references, explains results and decides the next step. The same Skill gives different people different plans.'
    ),
    emphasis: true,
  }
  const tools: FlowNode[] = [
    {
      label: t('State machine'),
      title: t('What to ask next'),
      body: t(
        'Over a hundred interview questions, scheduled by condition. Route questions only appear after you pick a route.'
      ),
    },
    {
      label: t('Scripts'),
      title: t('Run by you, or by the agent once approved'),
      body: t(
        'VM setup, firewall and egress checks run from documented scripts.'
      ),
    },
    {
      label: t('Verify'),
      title: t('Evidence and rollback'),
      body: t(
        'Each change records the original value, so it can be checked and reverted.'
      ),
    },
  ]

  return (
    <GuardianSection id='how-it-works'>
      <SectionHeading
        id='how-it-works'
        eyebrow={t('How it works')}
        title={t('The agent leads. The Skill keeps it honest.')}
        lede={t(
          'Claude-Guardian is not a one-click script. It gives an agent the context to understand your situation first, then propose a plan that fits.'
        )}
      />

      <div className='mt-14 grid items-center gap-4 lg:grid-cols-[1fr_48px_1.1fr_48px_1fr]'>
        <div className='grid gap-4'>
          {inputs.map((node, index) => (
            <AnimateInView key={node.label} delay={index * 80}>
              <FlowCard node={node} />
            </AnimateInView>
          ))}
        </div>
        <FlowConnector />
        <AnimateInView delay={160}>
          <FlowCard node={agent} />
        </AnimateInView>
        <FlowConnector />
        <div className='grid gap-4'>
          {tools.map((node, index) => (
            <AnimateInView key={node.label} delay={240 + index * 80}>
              <FlowCard node={node} />
            </AnimateInView>
          ))}
        </div>
      </div>
    </GuardianSection>
  )
}
