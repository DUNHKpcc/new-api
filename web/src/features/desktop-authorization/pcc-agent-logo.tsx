import { cn } from '@/lib/utils'

interface PccAgentLogoProps {
  className?: string
}

export function PccAgentLogo(props: PccAgentLogoProps) {
  return (
    <img
      src='/pcc-agent-logo.png'
      alt=''
      draggable={false}
      className={cn('shrink-0 select-none object-contain', props.className)}
    />
  )
}
