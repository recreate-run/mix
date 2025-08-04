import { Check, Clock, Circle } from 'lucide-react';

type TodoStatus = 'pending' | 'in_progress' | 'completed';
type TodoPriority = 'low' | 'medium' | 'high';

type Todo = {
  id: string;
  content: string;
  status: TodoStatus;
  priority: TodoPriority;
};

type TodoListProps = {
  todos: Todo[];
};

const StatusIcon = ({ status }: { status: TodoStatus }) => {
  switch (status) {
    case 'completed':
      return <Check className="size-4 text-green-500" />;
    case 'in_progress':
      return <Clock className="size-4 text-blue-500 animate-pulse" />;
    case 'pending':
      return <Circle className="size-4 text-gray-400" />;
  }
};


export function TodoList({ todos }: TodoListProps) {
  if (!todos.length) {
    return <div className="text-gray-500 text-sm">No todos</div>;
  }

  return (
    <div className="space-y-2">
      {todos.map((todo) => (
        <div
          key={todo.id}
          className="flex items-center gap-3 p-1"
        >
          <StatusIcon status={todo.status} />
          <div className="flex-1">
            <p className={`text-sm ${todo.status === 'completed' ? 'line-through text-gray-500' : 'text-gray-900 dark:text-gray-100'}`}>
              {todo.content}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}