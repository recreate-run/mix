import { AIResponse } from '@/components/ui/kibo-ui/ai/response';

interface HelpCommand {
  name: string;
  description: string;
  usage: string;
}

interface HelpData {
  type: string;
  commands: HelpCommand[];
}

interface HelpDisplayProps {
  data: HelpData;
}

export function HelpDisplay({ data }: HelpDisplayProps) {
  // Generate markdown string
  let markdown = '# Available Commands\n\n';

  data.commands.forEach((command) => {
    markdown += `- \`${command.usage}\` - ${command.description}\n`;
  });

  return <AIResponse>{markdown}</AIResponse>;
}
