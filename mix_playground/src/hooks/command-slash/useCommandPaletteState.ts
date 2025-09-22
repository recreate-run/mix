import { useState, useRef } from 'react';
import type {
  ViewState,
  AuthMethod,
  HierarchicalModelData,
  LoginData,
  LogoutData,
  StatusData,
  HelpData,
} from '@/types/command-slash';

export function useCommandPaletteState() {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [currentView, setCurrentView] = useState<ViewState>('commands');

  // Selection states
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [selectedMCPServer, setSelectedMCPServer] = useState<string | null>(null);
  const [selectedAuthMethod, setSelectedAuthMethod] = useState<AuthMethod | null>(null);

  // Command data states
  const [hierarchicalModelData, setHierarchicalModelData] = useState<HierarchicalModelData | undefined>(undefined);
  const [loginData, setLoginData] = useState<LoginData | undefined>(undefined);
  const [logoutData, setLogoutData] = useState<LogoutData | undefined>(undefined);
  const [statusData, setStatusData] = useState<StatusData | undefined>(undefined);
  const [helpData, setHelpData] = useState<HelpData | undefined>(undefined);

  // Refs for tracking initialization
  const hierarchicalModelInitializedRef = useRef(false);

  // Reset all command-related states
  const resetCommandStates = () => {
    setHierarchicalModelData(undefined);
    setLogoutData(undefined);
    setStatusData(undefined);
    setLoginData(undefined);
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
      case 'login-auth-input':
        setSelectedAuthMethod(null);
        setCurrentView('login-auth-methods');
        break;
      case 'login-auth-methods':
        setSelectedProvider(null);
        setCurrentView('login');
        break;
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
  const isShowingLogin = currentView.startsWith('login');
  const isShowingLoginProviders = currentView === 'login';
  const isShowingLoginAuthMethods = currentView === 'login-auth-methods';
  const isShowingLoginAuthInput = currentView === 'login-auth-input';
  const isShowingLogout = currentView === 'logout';
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
    loginData,
    logoutData,
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
    setLoginData,
    setLogoutData,
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
    isShowingLogin,
    isShowingLoginProviders,
    isShowingLoginAuthMethods,
    isShowingLoginAuthInput,
    isShowingLogout,
    isShowingStatus,
    isShowingHelp,
    isShowingCommands,
  };
}