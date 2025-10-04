import { useRef, useState } from 'react';
import type {
  AuthMethod,
  HelpData,
  HierarchicalModelData,
  StatusData,
  ViewState,
} from '@/types/command-slash';

export function useCommandPaletteState() {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [currentView, setCurrentView] = useState<ViewState>('commands');

  // Selection states
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [selectedMCPServer, setSelectedMCPServer] = useState<string | null>(
    null
  );
  const [selectedAuthMethod, setSelectedAuthMethod] =
    useState<AuthMethod | null>(null);

  // Command data states
  const [hierarchicalModelData, setHierarchicalModelData] = useState<
    HierarchicalModelData | undefined
  >(undefined);
  const [statusData, setStatusData] = useState<StatusData | undefined>(
    undefined
  );
  const [helpData, setHelpData] = useState<HelpData | undefined>(undefined);

  // Refs for tracking initialization
  const hierarchicalModelInitializedRef = useRef(false);

  // Reset all command-related states
  const resetCommandStates = () => {
    setHierarchicalModelData(undefined);
    setStatusData(undefined);
    setHelpData(undefined);
    setCurrentView('commands');
    setSelectedProvider(null);
    setSelectedMCPServer(null);
    setSelectedAuthMethod(null);
  };

  // View navigation helpers
  const goToView = (view: ViewState) => {
    setCurrentView(view);
  };

  const goBack = () => {
    switch (currentView) {
      case 'hierarchical-models':
        setSelectedProvider(null);
        setCurrentView('hierarchical-model');
        break;
      case 'mcp-tools':
        setSelectedMCPServer(null);
        setCurrentView('mcp');
        break;
      default:
        resetCommandStates();
        break;
    }
  };

  const resetToCommands = () => {
    resetCommandStates();
  };

  // Derived state
  const isShowingPermissions = currentView === 'permissions';
  const isShowingSessions = currentView === 'sessions';
  const isShowingMCP = currentView === 'mcp';
  const isShowingMCPTools = currentView === 'mcp-tools';
  const isShowingHierarchicalModel = currentView === 'hierarchical-model';
  const isShowingHierarchicalModels = currentView === 'hierarchical-models';
  const isShowingStatus = currentView === 'status';
  const isShowingHelp = currentView === 'help';
  const isShowingCommands = currentView === 'commands';

  return {
    // State values
    selectedValue,
    searchQuery,
    currentView,
    selectedProvider,
    selectedMCPServer,
    selectedAuthMethod,
    hierarchicalModelData,
    statusData,
    helpData,
    hierarchicalModelInitializedRef,

    // State setters
    setSelectedValue,
    setSearchQuery,
    setCurrentView,
    setSelectedProvider,
    setSelectedMCPServer,
    setSelectedAuthMethod,
    setHierarchicalModelData,
    setStatusData,
    setHelpData,

    // Navigation helpers
    goToView,
    goBack,
    resetToCommands,
    resetCommandStates,

    // Derived state
    isShowingPermissions,
    isShowingSessions,
    isShowingMCP,
    isShowingMCPTools,
    isShowingHierarchicalModel,
    isShowingHierarchicalModels,
    isShowingStatus,
    isShowingHelp,
    isShowingCommands,
  };
}
