import React, { useState, useEffect } from 'react';
import {
  Database,
  Search,
  FileText,
  Trash2,
  Plus,
  RefreshCw,
  ChevronRight,
  ChevronDown,
  Settings,
  Key,
  Filter,
  Download,
  Pencil,
} from 'lucide-react';
import * as backend from '../wailsjs/go/main/App';

export default function PebbleDBExplorer() {
  const [dbs, setDbs] = useState<string[]>([]);
  const [selectedDb, setSelectedDb] = useState<string>('');
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [viewMode, setViewMode] = useState<'tree' | 'list'>('tree');
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());
  const [keysByDb, setKeysByDb] = useState<{ [db: string]: { [prefix: string]: string[] } }>({});
  const [values, setValues] = useState<{ [cacheKey: string]: string }>({});
  const [valueMetadata, setValueMetadata] = useState<any>(null);
  const [viewFormat, setViewFormat] = useState<'auto' | 'hex' | 'base64' | 'string'>('auto');
  const [isAddingConnection, setIsAddingConnection] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [connectionName, setConnectionName] = useState('');
  const [connectionPath, setConnectionPath] = useState('');
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const [connectionPaths, setConnectionPaths] = useState<Record<string, string>>({});
  const [connectionToEdit, setConnectionToEdit] = useState<string | null>(null);
  const [dbStats, setDbStats] = useState<any>(null);
  const [isLoadingKeys, setIsLoadingKeys] = useState(false);
  const [loadProgress, setLoadProgress] = useState(0);

  const hasWailsBackend = () => Boolean((window as any).go?.main?.App);

  const safeGetDatabases = async () => {
    if (hasWailsBackend()) {
      const result = await backend.GetDatabases();
      return Array.isArray(result) ? result : [];
    }
    console.warn('Wails backend not available. Returning empty database list.');
    return [] as string[];
  };

  const safeGetConnectionPaths = async () => {
    if (hasWailsBackend()) {
      const result = await backend.GetConnectionPaths();
      return result ?? {};
    }
    console.warn('Wails backend not available. Returning empty connection paths.');
    return {} as Record<string, string>;
  };

  const safeGetKeysByPrefix = async (db: string) => {
    if (hasWailsBackend()) {
      return backend.GetKeysByPrefix(db);
    }
    console.warn('Wails backend not available. Skipping key fetch.');
    return null;
  };

  const safeGetValue = async (db: string, key: string) => {
    if (hasWailsBackend()) {
      return backend.GetValue(db, key);
    }
    console.warn('Wails backend not available. Returning empty value.');
    return '';
  };

  const safeAddConnection = async (name: string, path: string) => {
    if (!hasWailsBackend()) {
      throw new Error('Wails backend is not available. Run the desktop app to add connections.');
    }
    await backend.AddConnection(name, path);
  };

  const safeUpdateConnection = async (oldName: string, newName: string, path: string) => {
    if (!hasWailsBackend()) {
      throw new Error('Wails backend is not available. Run the desktop app to edit connections.');
    }
    await backend.UpdateConnection(oldName, newName, path);
  };

  const resetConnectionForm = () => {
    setConnectionName('');
    setConnectionPath('');
    setConnectionError(null);
  };

  const openConnectionForm = (name?: string) => {
    if (name) {
      setConnectionToEdit(name);
      setConnectionName(name);
      setConnectionPath(connectionPaths[name] ?? '');
      setConnectionError(null);
    } else {
      setConnectionToEdit(null);
      resetConnectionForm();
    }
    setIsAddingConnection(true);
  };

  const closeConnectionForm = () => {
    setIsAddingConnection(false);
    setConnectionToEdit(null);
    resetConnectionForm();
  };

  const handleBrowseForPath = async () => {
    if (!hasWailsBackend()) {
      setConnectionError('Directory picker is not available in this environment. Please paste the path manually.');
      return;
    }
    try {
      const selected = await backend.OpenDirectoryDialog();
      if (selected) {
        setConnectionPath(selected);
        setConnectionError(null);
      }
    } catch (err: any) {
      setConnectionError(err?.message ?? 'Failed to open directory picker');
    }
  };

  const submitConnection = async () => {
    const trimmedName = connectionName.trim();
    const trimmedPath = connectionPath.trim();
    const isEditing = connectionToEdit !== null;

    if (!trimmedName) {
      setConnectionError('Enter a connection name.');
      return;
    }
    if (!trimmedPath) {
      setConnectionError('Provide the PebbleDB directory path.');
      return;
    }

    const nameExists = dbs
      .filter(db => (isEditing ? db !== connectionToEdit : true))
      .some(db => db === trimmedName);

    if (nameExists) {
      setConnectionError('A connection with this name already exists.');
      return;
    }

    try {
      if (isEditing && connectionToEdit) {
        await safeUpdateConnection(connectionToEdit, trimmedName, trimmedPath);
      } else {
        await safeAddConnection(trimmedName, trimmedPath);
      }
      const newDbs: string[] = await safeGetDatabases();
      setDbs(newDbs);
      const newPaths = await safeGetConnectionPaths();
      setConnectionPaths(newPaths);
      setKeysByDb(prev => {
        const updated = { ...prev };
        delete updated[selectedDb];
        return updated;
      });
      setSelectedDb(trimmedName);
      closeConnectionForm();
    } catch (err: any) {
      setConnectionError(err?.message ?? String(err));
    }
  };

  useEffect(() => {
    safeGetDatabases().then((res: string[]) => setDbs(res ?? []));
    safeGetConnectionPaths().then(paths => setConnectionPaths(paths ?? {}));
  }, []);

  useEffect(() => {
    if (selectedDb && !keysByDb[selectedDb]) {
      setIsLoadingKeys(true);
      setLoadProgress(0);

      let progressInterval: any = null;

      // Get database stats first for progress calculation
      if (hasWailsBackend()) {
        backend.GetDatabaseStats(selectedDb).then((stats: any) => {
          console.log('Database stats:', stats);
          setDbStats(stats);

          // Simulate progress (since we can't get real progress from sync call)
          let progress = 0;
          progressInterval = setInterval(() => {
            progress += 3;
            if (progress >= 90) {
              if (progressInterval) clearInterval(progressInterval);
              setLoadProgress(90);
            } else {
              setLoadProgress(progress);
            }
          }, 150);

          // Now load keys with progress updates
          const startTime = Date.now();
          console.log(`Starting to load keys for database: ${selectedDb}`);

          safeGetKeysByPrefix(selectedDb).then((prefixes: { [prefix: string]: string[] } | null) => {
            console.log('GetKeysByPrefix returned:', prefixes);
            console.log('Type of prefixes:', typeof prefixes);
            console.log('Is array?', Array.isArray(prefixes));
            console.log('Is null?', prefixes === null);
            console.log('Is undefined?', prefixes === undefined);

            if (progressInterval) clearInterval(progressInterval);

            if (prefixes && typeof prefixes === 'object' && !Array.isArray(prefixes)) {
              const prefixCount = Object.keys(prefixes).length;
              const totalKeys = Object.values(prefixes).reduce((sum, arr) => sum + arr.length, 0);
              console.log(`✓ Loaded ${totalKeys} keys in ${prefixCount} prefixes in ${Date.now() - startTime}ms`);
              console.log('Sample prefixes:', Object.keys(prefixes).slice(0, 5));

              setKeysByDb(prev => {
                const updated = { ...prev, [selectedDb]: prefixes };
                console.log('Updated keysByDb:', updated);
                return updated;
              });
              setLoadProgress(100);

              // Keep at 100% for a moment before hiding
              setTimeout(() => {
                setIsLoadingKeys(false);
              }, 500);
            } else {
              console.error('❌ Invalid prefixes data:', prefixes);
              console.error('Expected object, got:', typeof prefixes);
              setIsLoadingKeys(false);
            }
          }).catch((err) => {
            if (progressInterval) clearInterval(progressInterval);
            console.error('❌ Error loading keys:', err);
            console.error('Error stack:', err?.stack);
            setIsLoadingKeys(false);
          });
        }).catch((err) => {
          console.error('Error getting stats:', err);
          setIsLoadingKeys(false);
        });
      } else {
        // No backend, just load
        safeGetKeysByPrefix(selectedDb).then((prefixes: { [prefix: string]: string[] } | null) => {
          if (prefixes) {
            console.log('Keys loaded (no backend):', prefixes);
            setKeysByDb(prev => ({ ...prev, [selectedDb]: prefixes }));
          }
          setIsLoadingKeys(false);
        }).catch((err) => {
          console.error('Error loading keys:', err);
          setIsLoadingKeys(false);
        });
      }

      // Cleanup on unmount
      return () => {
        if (progressInterval) clearInterval(progressInterval);
      };
    }
  }, [selectedDb]);

  useEffect(() => {
    if (selectedDb && selectedKey) {
      const cacheKey = `${selectedDb}_${selectedKey}`;
      if (!values[cacheKey]) {
        // Get value with metadata for better display
        if (hasWailsBackend()) {
          backend.GetValueWithMetadata(selectedDb, selectedKey).then((metadata: any) => {
            if (metadata) {
              setValueMetadata(metadata);
              setValues(prev => ({ ...prev, [cacheKey]: metadata.value }));
            }
          });
        } else {
          safeGetValue(selectedDb, selectedKey).then((val: string) => {
            if (val) {
              setValues(prev => ({ ...prev, [cacheKey]: val }));
            }
          });
        }
      }
    }
  }, [selectedDb, selectedKey]);

  const getKeysByPrefix = () => keysByDb[selectedDb] || {};

  const filteredKeys = () => {
    const prefixes = getKeysByPrefix();
    let allKeys: string[] = [];
    Object.values(prefixes).forEach(keys => {
      allKeys = allKeys.concat(keys);
    });
    return allKeys.filter(key => key.toLowerCase().includes(searchTerm.toLowerCase()));
  };

  const toggleExpand = (prefix: string) => {
    const newExpanded = new Set(expandedKeys);
    if (newExpanded.has(prefix)) {
      newExpanded.delete(prefix);
    } else {
      newExpanded.add(prefix);
    }
    setExpandedKeys(newExpanded);
  };

  const handleRefresh = () => {
    if (selectedDb) {
      safeGetKeysByPrefix(selectedDb).then((prefixes: { [prefix: string]: string[] } | null) => {
        if (prefixes) {
          setKeysByDb(prev => ({ ...prev, [selectedDb]: prefixes }));
        }
      });
    }
  };

  const handleExport = async () => {
    if (!selectedDb || !selectedKey || !hasWailsBackend()) return;

    setIsExporting(true);
    try {
      await backend.ExportValue(selectedDb, selectedKey);
      // Success - file saved
    } catch (err: any) {
      alert(`Export failed: ${err?.message || String(err)}`);
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="relative flex h-screen bg-slate-100 text-slate-900">
      <div className="w-64 bg-[#19334D] text-slate-100 flex flex-col">
        <div className="p-4 border-b border-slate-700/60">
          <div className="flex items-center gap-2 mb-4">
            <Database className="w-6 h-6 text-slate-200" />
            <h1 className="text-lg font-semibold tracking-wide">GPaw Explorer</h1>
          </div>
          <button
            onClick={() => openConnectionForm()}
            className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-[#003C67] px-4 py-2 font-medium text-white transition hover:opacity-90"
          >
            <Plus className="w-4 h-4" />
            New Connection
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          <div className="px-2 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Databases</div>
          {dbs.map(db => (
            <div
              key={db}
              onClick={() => setSelectedDb(db)}
              className={`mb-1 flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                selectedDb === db
                  ? 'bg-[#003C67] text-white shadow-inner'
                  : 'hover:bg-slate-800/40'
              }`}
            >
              <Database className="w-4 h-4" />
              <span className="flex-1 truncate">{db}</span>
              <span className="text-xs text-slate-300">
                {Object.keys(keysByDb[db] || {}).reduce((sum, p) => sum + (keysByDb[db][p]?.length || 0), 0)}
              </span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  openConnectionForm(db);
                }}
                className="ml-1 rounded-md p-1 text-slate-200 transition hover:bg-slate-800/60"
                title="Edit connection"
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
          {dbs.length === 0 && (
            <div className="px-3 py-2 text-sm text-slate-300">
              No databases connected. Add a new connection.
            </div>
          )}
        </div>

        <div className="border-t border-slate-700/60 p-4">
          <button className="w-full inline-flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition hover:bg-slate-800/50">
            <Settings className="w-4 h-4" />
            Settings
          </button>
        </div>
      </div>

      <div className="flex flex-1 flex-col bg-white">
        <div className="flex items-center gap-3 border-b border-slate-200 bg-white p-4">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              placeholder="Search keys..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white py-2 pl-10 pr-4 text-sm shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500"
            />
          </div>
          <button className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-sky-100 px-4 py-2 text-sm font-medium text-sky-800 transition hover:bg-sky-200">
            <Filter className="w-4 h-4" />
            Filter
          </button>
          <button
            onClick={handleRefresh}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-sky-100 px-4 py-2 text-sm font-medium text-sky-800 transition hover:bg-sky-200"
          >
            <RefreshCw className="w-4 h-4" />
            Refresh
          </button>
          <button className="inline-flex items-center gap-2 rounded-lg bg-[#003C67] px-4 py-2 text-sm font-medium text-white transition hover:opacity-90">
            <Plus className="w-4 h-4" />
            Add Key
          </button>
        </div>

        <div className="flex flex-1 min-h-0">
          <div className="flex w-80 flex-col border-r border-slate-200 bg-white">
            <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 text-sm font-medium text-slate-600">
              <span>
                Keys ({viewMode === 'tree'
                  ? Object.keys(getKeysByPrefix()).reduce((sum, p) => sum + getKeysByPrefix()[p].length, 0)
                  : filteredKeys().length})
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setViewMode('tree')}
                  className={`rounded-md px-3 py-1 text-xs font-semibold transition ${
                    viewMode === 'tree'
                      ? 'bg-sky-100 text-sky-800'
                      : 'text-slate-500 hover:bg-slate-100'
                  }`}
                >
                  Tree
                </button>
                <button
                  onClick={() => setViewMode('list')}
                  className={`rounded-md px-3 py-1 text-xs font-semibold transition ${
                    viewMode === 'list'
                      ? 'bg-sky-100 text-sky-800'
                      : 'text-slate-500 hover:bg-slate-100'
                  }`}
                >
                  List
                </button>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto p-3 space-y-2">
              {dbStats && selectedDb && !isLoadingKeys && (
                <div className="mb-3 rounded-lg bg-blue-50 p-3 text-xs">
                  <div className="font-semibold text-blue-900 mb-1">Database Stats:</div>
                  <div className="text-blue-700">Total Keys: {dbStats.totalKeys?.toLocaleString() || 0}</div>
                  {dbStats.firstKey && <div className="text-blue-700 truncate">First: {dbStats.firstKey}</div>}
                  {dbStats.error && <div className="text-red-600">Error: {dbStats.error}</div>}
                  {selectedDb && keysByDb[selectedDb] && (
                    <div className="text-green-700 mt-1">
                      ✓ Loaded: {Object.keys(keysByDb[selectedDb]).length} prefixes
                    </div>
                  )}
                </div>
              )}
              {isLoadingKeys && (
                <div className="mb-3 rounded-lg bg-blue-50 border border-blue-200 p-4">
                  <div className="flex items-center gap-3 mb-2">
                    <svg className="animate-spin h-5 w-5 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    <span className="text-sm font-medium text-blue-900">
                      Loading keys... {dbStats?.totalKeys ? `(~${dbStats.totalKeys.toLocaleString()} total)` : ''}
                    </span>
                  </div>
                  <div className="w-full bg-blue-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{ width: `${loadProgress}%` }}
                    ></div>
                  </div>
                  <div className="text-xs text-blue-700 mt-1 text-right">{loadProgress}%</div>
                </div>
              )}
              {!isLoadingKeys && selectedDb && Object.keys(getKeysByPrefix()).length === 0 && (
                <div className="mb-3 rounded-lg bg-yellow-50 border border-yellow-200 p-4 text-center">
                  <p className="text-sm text-yellow-800">No keys found in this database.</p>
                  <p className="text-xs text-yellow-600 mt-1">Check console for errors or try refreshing.</p>
                </div>
              )}
              {viewMode === 'tree'
                ? Object.entries(getKeysByPrefix()).map(([prefix, keys]) => (
                    <div key={prefix} className="space-y-1">
                      <div
                        onClick={() => toggleExpand(prefix)}
                        className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm font-medium text-slate-600 transition hover:bg-sky-100"
                      >
                        {expandedKeys.has(prefix) ? (
                          <ChevronDown className="h-4 w-4 text-slate-500" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-slate-500" />
                        )}
                        <Key className="h-4 w-4 text-sky-700" />
                        <span>{prefix || 'default'}</span>
                        <span className="ml-auto text-xs text-slate-400">{keys.length}</span>
                      </div>
                      {expandedKeys.has(prefix) && (
                        <div className="ml-6 space-y-1">
                          {keys.map(key => (
                            <div
                              key={key}
                              onClick={() => setSelectedKey(key)}
                              className={`flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm transition ${
                                selectedKey === key
                                  ? 'bg-sky-100 text-sky-800'
                                  : 'text-slate-600 hover:bg-slate-100'
                              }`}
                            >
                              <FileText className="h-4 w-4" />
                              <span className="truncate">{key}</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ))
                : filteredKeys().map(key => (
                    <div
                      key={key}
                      onClick={() => setSelectedKey(key)}
                      className={`flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm transition ${
                        selectedKey === key
                          ? 'bg-sky-100 text-sky-800'
                          : 'text-slate-600 hover:bg-slate-100'
                      }`}
                    >
                      <Key className="h-4 w-4" />
                      <span className="truncate">{key}</span>
                    </div>
                  ))}
            </div>
          </div>

          <div className="flex-1 overflow-hidden bg-slate-50 flex flex-col">
            {selectedKey ? (
              <div className="flex flex-col h-full">
                {/* Header */}
                <div className="flex-none bg-white border-b border-slate-200 px-6 py-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <h2 className="text-lg font-semibold text-slate-900 break-all">{selectedKey}</h2>
                      {valueMetadata && (
                        <div className="flex items-center gap-4 mt-2 text-sm text-slate-600">
                          <span className="flex items-center gap-1">
                            <span className="text-slate-500">Type:</span>
                            <span className="font-mono text-sky-700 capitalize">{valueMetadata.type}</span>
                          </span>
                          <span className="flex items-center gap-1">
                            <span className="text-slate-500">Size:</span>
                            <span className="font-mono">{valueMetadata.size.toLocaleString()} bytes</span>
                          </span>
                        </div>
                      )}
                    </div>
                    <div className="flex-none flex gap-2">
                      <button
                        onClick={handleExport}
                        disabled={isExporting}
                        className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm transition hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        <Download className="h-4 w-4" />
                        {isExporting ? 'Exporting...' : 'Export'}
                      </button>
                      <button className="inline-flex items-center gap-2 rounded-md border border-rose-400 bg-rose-50 px-3 py-1.5 text-sm text-rose-600 transition hover:bg-rose-100">
                        <Trash2 className="h-4 w-4" />
                        Delete
                      </button>
                    </div>
                  </div>
                </div>

                {/* Format Selector */}
                <div className="flex-none bg-white border-b border-slate-200 px-6 py-3">
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-medium text-slate-700">View as:</span>
                    <div className="flex gap-2">
                      <button
                        onClick={() => setViewFormat('auto')}
                        className={`rounded-md px-3 py-1.5 text-xs font-semibold transition ${
                          viewFormat === 'auto' ? 'bg-sky-100 text-sky-800 ring-1 ring-sky-300' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                        }`}
                      >
                        Auto
                      </button>
                      <button
                        onClick={() => setViewFormat('string')}
                        className={`rounded-md px-3 py-1.5 text-xs font-semibold transition ${
                          viewFormat === 'string' ? 'bg-sky-100 text-sky-800 ring-1 ring-sky-300' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                        }`}
                      >
                        String
                      </button>
                      <button
                        onClick={() => setViewFormat('hex')}
                        className={`rounded-md px-3 py-1.5 text-xs font-semibold transition ${
                          viewFormat === 'hex' ? 'bg-sky-100 text-sky-800 ring-1 ring-sky-300' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                        }`}
                      >
                        Hex
                      </button>
                      <button
                        onClick={() => setViewFormat('base64')}
                        className={`rounded-md px-3 py-1.5 text-xs font-semibold transition ${
                          viewFormat === 'base64' ? 'bg-sky-100 text-sky-800 ring-1 ring-sky-300' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                        }`}
                      >
                        Base64
                      </button>
                    </div>
                  </div>
                </div>

                {/* Value Display */}
                <div className="flex-1 overflow-auto p-6">
                  {valueMetadata?.isTruncated && (
                    <div className="mb-4 rounded-lg bg-amber-50 border border-amber-200 p-4">
                      <div className="flex items-start gap-3">
                        <svg className="w-5 h-5 text-amber-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                        </svg>
                        <div className="flex-1">
                          <h4 className="text-sm font-semibold text-amber-900 mb-1">Large Value Detected</h4>
                          <p className="text-sm text-amber-800">{valueMetadata.truncatedMsg}</p>
                        </div>
                      </div>
                    </div>
                  )}
                  <div className="rounded-lg bg-[#19334D] p-4 shadow-lg h-full min-h-[400px]">
                    <pre className="text-sm font-mono text-emerald-300 whitespace-pre-wrap break-all">
                      {(() => {
                        if (!valueMetadata) return 'Loading...';

                        // Determine which format to display
                        let displayValue = '';
                        switch (viewFormat) {
                          case 'hex':
                            displayValue = valueMetadata.valueHex;
                            break;
                          case 'base64':
                            displayValue = valueMetadata.valueBase64;
                            break;
                          case 'string':
                            displayValue = valueMetadata.value;
                            break;
                          case 'auto':
                          default:
                            // Auto-detect best format
                            if (valueMetadata.type === 'json') {
                              displayValue = valueMetadata.value; // Already pretty-printed
                            } else if (valueMetadata.type === 'string') {
                              displayValue = valueMetadata.value;
                            } else {
                              // Binary data - show hex
                              displayValue = valueMetadata.valueHex;
                            }
                            break;
                        }

                        return displayValue;
                      })()}
                    </pre>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center text-slate-400">
                <div className="text-center space-y-3">
                  <Database className="mx-auto h-20 w-20 opacity-30" />
                  <p className="text-lg font-medium text-slate-500">Select a key to view its value</p>
                  <p className="text-sm text-slate-400">Choose from the list on the left</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {isAddingConnection && (
        <div className="absolute inset-0 z-40 flex items-center justify-center bg-slate-900/70 px-4">
          <div className="w-full max-w-xl rounded-2xl bg-white p-6 shadow-2xl">
            <div className="mb-4">
              <h2 className="text-xl font-semibold text-slate-900">
                {connectionToEdit ? 'Edit Connection' : 'Add PebbleDB Connection'}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                Paste the database directory path or browse to it. Example:
                <code className="ml-1 rounded bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                  /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db
                </code>
              </p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-slate-700">Connection Name</label>
                <input
                  value={connectionName}
                  onChange={(e) => setConnectionName(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500"
                  placeholder="e.g. slot-db"
                />
              </div>

              <div>
                <label className="text-sm font-medium text-slate-700">Database Path</label>
                <div className="mt-1 flex gap-2">
                  <input
                    value={connectionPath}
                    onChange={(e) => setConnectionPath(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500"
                    placeholder="/path/to/pebbledb"
                  />
                  <button
                    onClick={handleBrowseForPath}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100"
                  >
                    Browse
                  </button>
                </div>
                {connectionPath && (
                  <p className="mt-2 text-xs text-slate-500">
                    Selected path: <span className="font-mono text-slate-600">{connectionPath}</span>
                  </p>
                )}
              </div>

              {connectionError && (
                <div className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">
                  {connectionError}
                </div>
              )}

              <div className="flex justify-end gap-3">
                <button
                  onClick={closeConnectionForm}
                  className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100"
                >
                  Cancel
                </button>
                <button
                  onClick={submitConnection}
                  className="rounded-lg bg-[#003C67] px-4 py-2 text-sm font-semibold text-white transition hover:opacity-90"
                >
                  {connectionToEdit ? 'Save Changes' : 'Connect'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
