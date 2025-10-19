"use client";

import { useState, useRef, useEffect } from "react";
import Image from "next/image";
import { Tree, NodeRendererProps, TreeApi, NodeApi } from "react-arborist";
import { useTranslations } from "next-intl";
import { useTheme } from "next-themes";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  ChevronRight,
  File,
  Folder,
  FolderOpen,
  Loader2,
  Download,
  Plus,
  Trash2,
  RefreshCw,
  Upload,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  getArtifacts,
  getListFiles,
  getFile,
  createArtifact,
  deleteArtifact,
  uploadFile,
  deleteFile,
} from "@/api/models/artifact";
import { Artifact, ListFilesResp, File as FileInfo } from "@/types";
import ReactCodeMirror from "@uiw/react-codemirror";
import { okaidia } from "@uiw/codemirror-theme-okaidia";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";

interface TreeNode {
  id: string;
  name: string;
  type: "folder" | "file";
  path: string;
  children?: TreeNode[];
  isLoaded?: boolean;
  fileInfo?: FileInfo; // Store complete file information
}

interface NodeProps extends NodeRendererProps<TreeNode> {
  loadingNodes: Set<string>;
  onUploadClick: (path: string) => void;
  isUploading: boolean;
  t: (key: string) => string;
}

function truncateMiddle(str: string, maxLength: number = 30): string {
  if (str.length <= maxLength) return str;

  const ellipsis = "...";
  const charsToShow = maxLength - ellipsis.length;
  const frontChars = Math.ceil(charsToShow / 2);
  const backChars = Math.floor(charsToShow / 2);

  return (
    str.substring(0, frontChars) +
    ellipsis +
    str.substring(str.length - backChars)
  );
}

function Node({
  node,
  style,
  dragHandle,
  loadingNodes,
  onUploadClick,
  isUploading,
  t,
}: NodeProps) {
  const indent = node.level * 12;
  const isFolder = node.data.type === "folder";
  const isLoading = loadingNodes.has(node.id);
  const textRef = useRef<HTMLSpanElement>(null);
  const [displayName, setDisplayName] = useState(node.data.name);
  const [showUploadButton, setShowUploadButton] = useState(false);

  useEffect(() => {
    const updateDisplayName = () => {
      if (!textRef.current) return;

      const container = textRef.current.parentElement;
      if (!container) return;

      // Get available width (container width - icon width - gap - padding)
      const containerWidth = container.clientWidth;
      const iconWidth = isFolder ? 56 : 40; // Total width of icon and spacing
      const availableWidth = containerWidth - iconWidth;

      // Create temporary element to measure text width
      const tempSpan = document.createElement("span");
      tempSpan.style.visibility = "hidden";
      tempSpan.style.position = "absolute";
      tempSpan.style.fontSize = "14px"; // text-sm
      tempSpan.style.fontFamily = getComputedStyle(textRef.current).fontFamily;
      tempSpan.textContent = node.data.name;
      document.body.appendChild(tempSpan);

      const fullWidth = tempSpan.offsetWidth;
      document.body.removeChild(tempSpan);

      // If text width is less than available width, display full name
      if (fullWidth <= availableWidth) {
        setDisplayName(node.data.name);
        return;
      }

      // Calculate number of characters to display
      const charWidth = fullWidth / node.data.name.length;
      const maxChars = Math.floor(availableWidth / charWidth);

      setDisplayName(truncateMiddle(node.data.name, Math.max(10, maxChars)));
    };

    updateDisplayName();

    // Add window resize listener
    const resizeObserver = new ResizeObserver(updateDisplayName);
    if (textRef.current?.parentElement) {
      resizeObserver.observe(textRef.current.parentElement);
    }

    return () => {
      resizeObserver.disconnect();
    };
  }, [node.data.name, indent, isFolder]);

  return (
    <div
      ref={dragHandle}
      style={style}
      className={cn(
        "flex items-center cursor-pointer px-2 py-1.5 text-sm rounded-md transition-colors group",
        "hover:bg-accent hover:text-accent-foreground",
        node.isSelected && "bg-accent text-accent-foreground",
        node.state.isDragging && "opacity-50"
      )}
      onMouseEnter={() => isFolder && setShowUploadButton(true)}
      onMouseLeave={() => setShowUploadButton(false)}
      onClick={() => {
        if (isFolder) {
          node.toggle();
        } else {
          node.select();
        }
      }}
    >
      <div
        style={{ marginLeft: `${indent}px` }}
        className="flex items-center gap-1.5 flex-1 min-w-0"
      >
        {isFolder ? (
          <>
            {isLoading ? (
              <Loader2 className="h-4 w-4 shrink-0 animate-spin text-muted-foreground" />
            ) : (
              <ChevronRight
                className={cn(
                  "h-4 w-4 shrink-0 transition-transform duration-200",
                  node.isOpen && "rotate-90"
                )}
              />
            )}
            {node.isOpen ? (
              <FolderOpen className="h-4 w-4 shrink-0 text-muted-foreground" />
            ) : (
              <Folder className="h-4 w-4 shrink-0 text-muted-foreground" />
            )}
          </>
        ) : (
          <>
            <span className="w-4" />
            <File className="h-4 w-4 shrink-0 text-muted-foreground" />
          </>
        )}
        <span ref={textRef} className="min-w-0" title={node.data.name}>
          {displayName}
        </span>
      </div>
      {isFolder && showUploadButton && (
        <button
          className="shrink-0 ml-2 p-1 rounded-md bg-primary/10 hover:bg-primary/20 transition-colors"
          onClick={(e) => {
            e.stopPropagation();
            onUploadClick(node.data.path);
          }}
          disabled={isUploading}
          title={t("uploadToFolderTooltip")}
        >
          {isUploading ? (
            <Loader2 className="h-3 w-3 animate-spin text-primary" />
          ) : (
            <Upload className="h-3 w-3 text-primary" />
          )}
        </button>
      )}
    </div>
  );
}

export default function ArtifactPage() {
  const t = useTranslations("artifact");
  const { resolvedTheme } = useTheme();

  const treeRef = useRef<TreeApi<TreeNode>>(null);
  const [selectedFile, setSelectedFile] = useState<TreeNode | null>(null);
  const [loadingNodes, setLoadingNodes] = useState<Set<string>>(new Set());
  const [treeData, setTreeData] = useState<TreeNode[]>([]);
  const [isInitialLoading, setIsInitialLoading] = useState(false);

  // Artifact related states
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState<Artifact | null>(
    null
  );
  const [isLoadingArtifacts, setIsLoadingArtifacts] = useState(true);

  // File preview states
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [isLoadingPreview, setIsLoadingPreview] = useState(false);
  const [isLoadingDownload, setIsLoadingDownload] = useState(false);

  // Delete confirmation dialog states
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [artifactToDelete, setArtifactToDelete] = useState<Artifact | null>(
    null
  );
  const [isDeleting, setIsDeleting] = useState(false);

  // Delete file confirmation dialog states
  const [deleteFileDialogOpen, setDeleteFileDialogOpen] = useState(false);
  const [fileToDelete, setFileToDelete] = useState<TreeNode | null>(null);
  const [isDeletingFile, setIsDeletingFile] = useState(false);

  // Upload file states
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);

  // Upload dialog states
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [uploadPath, setUploadPath] = useState<string>("/");
  const [initialUploadPath, setInitialUploadPath] = useState<string>("/"); // Track the initial path clicked
  const [uploadMetaValue, setUploadMetaValue] = useState<string>("{}");
  const [uploadMetaError, setUploadMetaError] = useState<string>("");
  const [isUploadMetaValid, setIsUploadMetaValid] = useState(true);
  const [selectedUploadFile, setSelectedUploadFile] = useState<File | null>(
    null
  );

  // Create artifact states
  const [isCreating, setIsCreating] = useState(false);

  // Refresh states
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Filter state
  const [filterText, setFilterText] = useState("");

  // Filtered artifacts based on search text
  const filteredArtifacts = artifacts.filter((artifact) =>
    artifact.id.toLowerCase().includes(filterText.toLowerCase())
  );

  // Load artifacts function (extracted for reuse)
  const loadArtifacts = async () => {
    try {
      setIsLoadingArtifacts(true);
      const res = await getArtifacts();
      if (res.code !== 0) {
        console.error(res.message);
        return;
      }
      setArtifacts(res.data || []);
    } catch (error) {
      console.error("Failed to load artifacts:", error);
    } finally {
      setIsLoadingArtifacts(false);
    }
  };

  // Load artifact list when component mounts
  useEffect(() => {
    loadArtifacts();
  }, []);

  const formatFiles = (path: string, res: ListFilesResp) => {
    const files: TreeNode[] = res.files.map((file) => ({
      id: `${file.path}${file.filename}`,
      name: file.filename,
      type: "file",
      path: file.path,
      isLoaded: false,
      fileInfo: file,
    }));
    const directories: TreeNode[] = res.directories.map((directory) => ({
      id: `${path}${directory}/`,
      name: directory,
      type: "folder",
      path: `${path}${directory}/`,
      isLoaded: false,
    }));
    return [...directories, ...files];
  };

  // Load root directory files when artifact is selected
  const handleArtifactSelect = async (artifact: Artifact) => {
    setSelectedArtifact(artifact);
    setTreeData([]);
    setSelectedFile(null);

    try {
      setIsInitialLoading(true);
      const res = await getListFiles(artifact.id, "/");
      if (res.code !== 0 || !res.data) {
        console.error(res.message);
        return;
      }
      setTreeData(formatFiles("/", res.data));
    } catch (error) {
      console.error("Failed to load files:", error);
    } finally {
      setIsInitialLoading(false);
    }
  };

  const handleToggle = async (nodeId: string) => {
    const node = treeRef.current?.get(nodeId);
    if (!node || node.data.type !== "folder" || !selectedArtifact) return;

    // Return if already loaded
    if (node.data.isLoaded) return;

    // Mark as loading
    setLoadingNodes((prev) => new Set(prev).add(nodeId));

    try {
      // Load children using unified interface with artifact_id and path
      const children = await getListFiles(selectedArtifact.id, node.data.path);
      if (children.code !== 0 || !children.data) {
        console.error(children.message);
        return;
      }
      const files = formatFiles(node.data.path, children.data);

      // Update node data
      setTreeData((prevData) => {
        const updateNode = (nodes: TreeNode[]): TreeNode[] => {
          return nodes.map((n) => {
            if (n.id === nodeId) {
              return {
                ...n,
                children: files,
                isLoaded: true,
              };
            }
            if (n.children) {
              return {
                ...n,
                children: updateNode(n.children),
              };
            }
            return n;
          });
        };
        return updateNode(prevData);
      });
    } catch (error) {
      console.error("Failed to load children:", error);
    } finally {
      // Remove loading state
      setLoadingNodes((prev) => {
        const next = new Set(prev);
        next.delete(nodeId);
        return next;
      });
    }
  };

  const handleSelect = (nodes: NodeApi<TreeNode>[]) => {
    const node = nodes[0];
    if (node && node.data.type === "file") {
      setSelectedFile(node.data);
    }
  };

  // Handle create artifact
  const handleCreateArtifact = async () => {
    try {
      setIsCreating(true);
      const res = await createArtifact();
      if (res.code !== 0) {
        console.error(res.message);
        return;
      }
      // Reload artifacts list
      await loadArtifacts();
      // Auto-select the newly created artifact
      if (res.data) {
        setSelectedArtifact(res.data);
        handleArtifactSelect(res.data);
      }
    } catch (error) {
      console.error("Failed to create artifact:", error);
    } finally {
      setIsCreating(false);
    }
  };

  // Handle delete artifact confirmation
  const handleDeleteClick = (artifact: Artifact, e: React.MouseEvent) => {
    e.stopPropagation();
    setArtifactToDelete(artifact);
    setDeleteDialogOpen(true);
  };

  // Handle delete artifact
  const handleDeleteArtifact = async () => {
    if (!artifactToDelete) return;

    try {
      setIsDeleting(true);
      const res = await deleteArtifact(artifactToDelete.id);
      if (res.code !== 0) {
        console.error(res.message);
        return;
      }

      // Clear file selection and preview
      setSelectedFile(null);
      setImageUrl(null);

      // If the deleted artifact is the currently selected one, clear selection
      if (selectedArtifact?.id === artifactToDelete.id) {
        setSelectedArtifact(null);
        setTreeData([]);
      }

      // Reload artifacts list
      await loadArtifacts();

      // If there's a selected artifact (and it's not the one being deleted), reload its file tree
      if (selectedArtifact && selectedArtifact.id !== artifactToDelete.id) {
        setTreeData([]);
        const filesRes = await getListFiles(selectedArtifact.id, "/");
        if (filesRes.code === 0 && filesRes.data) {
          setTreeData(formatFiles("/", filesRes.data));
        }
      }
    } catch (error) {
      console.error("Failed to delete artifact:", error);
    } finally {
      setIsDeleting(false);
      setDeleteDialogOpen(false);
      setArtifactToDelete(null);
    }
  };

  // Handle refresh artifacts
  const handleRefreshArtifacts = async () => {
    try {
      setIsRefreshing(true);
      // Clear file selection and preview
      setSelectedFile(null);
      setImageUrl(null);

      // Reload artifacts list
      await loadArtifacts();

      // If there's a selected artifact, reload its file tree
      if (selectedArtifact) {
        setTreeData([]);
        const res = await getListFiles(selectedArtifact.id, "/");
        if (res.code === 0 && res.data) {
          setTreeData(formatFiles("/", res.data));
        }
      }
    } catch (error) {
      console.error("Failed to refresh artifacts:", error);
    } finally {
      setIsRefreshing(false);
    }
  };

  const validateJSON = (value: string): boolean => {
    const trimmed = value.trim();
    if (!trimmed) return false;
    try {
      JSON.parse(trimmed);
      return true;
    } catch {
      return false;
    }
  };

  const handleUploadMetaChange = (value: string) => {
    setUploadMetaValue(value);
    const isValid = validateJSON(value);
    setIsUploadMetaValid(isValid);
    if (!isValid && value.trim()) {
      try {
        JSON.parse(value.trim());
      } catch (error) {
        if (error instanceof SyntaxError) {
          setUploadMetaError(t("invalidJson") + ": " + error.message);
        }
      }
    } else {
      setUploadMetaError("");
    }
  };

  // Handle upload file button click
  const handleUploadClick = (path: string = "/") => {
    setUploadPath(path);
    setInitialUploadPath(path);
    setUploadMetaValue("{}");
    setUploadMetaError("");
    setIsUploadMetaValid(true);
    setSelectedUploadFile(null);
    fileInputRef.current?.click();
  };

  // Handle file selection (open dialog)
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    const file = files[0];
    setSelectedUploadFile(file);
    setUploadDialogOpen(true);
  };

  // Handle actual file upload
  const handleUploadConfirm = async () => {
    if (!selectedUploadFile || !selectedArtifact) return;

    // Parse and validate JSON
    let meta: Record<string, string> | undefined;
    const trimmedMetaValue = uploadMetaValue.trim();

    if (trimmedMetaValue && trimmedMetaValue !== "{}") {
      try {
        meta = JSON.parse(trimmedMetaValue);
        setUploadMetaError("");
      } catch (error) {
        if (error instanceof SyntaxError) {
          setUploadMetaError(t("invalidJson") + ": " + error.message);
        } else {
          setUploadMetaError(String(error));
        }
        return;
      }
    }

    try {
      setIsUploading(true);
      setUploadDialogOpen(false);

      const res = await uploadFile(
        selectedArtifact.id,
        uploadPath,
        selectedUploadFile,
        meta
      );

      if (res.code !== 0) {
        console.error(res.message);
        return;
      }

      // Refresh the entire directory tree from root
      setTreeData([]);
      const filesRes = await getListFiles(selectedArtifact.id, "/");
      if (filesRes.code === 0 && filesRes.data) {
        setTreeData(formatFiles("/", filesRes.data));
      }
    } catch (error) {
      console.error("Failed to upload file:", error);
    } finally {
      setIsUploading(false);
      setSelectedUploadFile(null);
      setUploadMetaValue("{}");
      setUploadMetaError("");
      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  // Handle cancel upload
  const handleUploadCancel = () => {
    setUploadDialogOpen(false);
    setSelectedUploadFile(null);
    setUploadMetaValue("{}");
    setUploadMetaError("");
    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  // Handle delete file click
  const handleDeleteFileClick = () => {
    if (!selectedFile) return;
    setFileToDelete(selectedFile);
    setDeleteFileDialogOpen(true);
  };

  // Handle delete file confirmation
  const handleDeleteFile = async () => {
    if (!fileToDelete || !selectedArtifact || !fileToDelete.fileInfo) return;

    try {
      setIsDeletingFile(true);
      const fullPath = `${fileToDelete.path}${fileToDelete.fileInfo.filename}`;
      const res = await deleteFile(selectedArtifact.id, fullPath);

      if (res.code !== 0) {
        console.error(res.message);
        return;
      }

      // Clear selected file
      setSelectedFile(null);

      // Helper function to get parent path
      const getParentPath = (path: string): string | null => {
        if (path === "/") return null;
        // Remove trailing slash if exists
        const normalizedPath = path.endsWith("/") ? path.slice(0, -1) : path;
        const lastSlashIndex = normalizedPath.lastIndexOf("/");
        if (lastSlashIndex <= 0) return "/";
        return normalizedPath.substring(0, lastSlashIndex + 1);
      };

      // Recursive function to reload directory and check if empty
      const reloadDirectoryRecursively = async (
        currentPath: string
      ): Promise<void> => {
        const filesRes = await getListFiles(selectedArtifact.id, currentPath);

        if (filesRes.code !== 0 || !filesRes.data) {
          console.error(filesRes.message);
          return;
        }

        const files = formatFiles(currentPath, filesRes.data);
        const isEmpty = files.length === 0;

        if (currentPath === "/") {
          // Update root directory
          setTreeData(files);
          return;
        }

        // Update the tree data
        setTreeData((prevData) => {
          const updateNode = (nodes: TreeNode[]): TreeNode[] => {
            return nodes.map((n) => {
              if (n.path === currentPath) {
                return {
                  ...n,
                  children: files,
                  isLoaded: true,
                };
              }
              if (n.children) {
                return {
                  ...n,
                  children: updateNode(n.children),
                };
              }
              return n;
            });
          };
          return updateNode(prevData);
        });

        // If directory is empty, recursively reload parent directory
        if (isEmpty) {
          const parentPath = getParentPath(currentPath);
          if (parentPath !== null) {
            await reloadDirectoryRecursively(parentPath);
          }
        }
      };

      // Start reloading from the parent path
      const parentPath = fileToDelete.path;
      await reloadDirectoryRecursively(parentPath);
    } catch (error) {
      console.error("Failed to delete file:", error);
    } finally {
      setIsDeletingFile(false);
      setDeleteFileDialogOpen(false);
      setFileToDelete(null);
    }
  };

  // Reset preview states when file selection changes
  useEffect(() => {
    setImageUrl(null);
  }, [selectedFile]);

  // Handle preview button click
  const handlePreviewClick = async () => {
    if (!selectedFile || !selectedArtifact || !selectedFile.fileInfo) return;

    try {
      setIsLoadingPreview(true);
      const res = await getFile(
        selectedArtifact.id,
        `${selectedFile.path}${selectedFile.fileInfo.filename}`
      );
      if (res.code !== 0) {
        console.error(res.message);
        return;
      }
      setImageUrl(res.data?.public_url || null);
    } catch (error) {
      console.error("Failed to load preview:", error);
    } finally {
      setIsLoadingPreview(false);
    }
  };

  // Handle download button click
  const handleDownloadClick = async () => {
    if (!selectedFile || !selectedArtifact || !selectedFile.fileInfo) return;

    try {
      setIsLoadingDownload(true);
      const res = await getFile(
        selectedArtifact.id,
        `${selectedFile.path}${selectedFile.fileInfo.filename}`
      );
      if (res.code !== 0) {
        console.error(res.message);
        return;
      }

      const downloadUrl = res.data?.public_url;
      if (downloadUrl) {
        const link = document.createElement("a");
        link.href = downloadUrl;
        link.download = selectedFile.fileInfo.filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      }
    } catch (error) {
      console.error("Failed to download file:", error);
    } finally {
      setIsLoadingDownload(false);
    }
  };

  return (
    <ResizablePanelGroup direction="horizontal">
      {/* Artifact List Panel */}
      <ResizablePanel defaultSize={25} minSize={15} maxSize={35}>
        <div className="h-full bg-background p-4 flex flex-col">
          <div className="mb-4 space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">{t("title")}</h2>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="icon"
                  onClick={handleCreateArtifact}
                  disabled={isCreating || isLoadingArtifacts}
                  title={t("createTooltip")}
                >
                  {isCreating ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                </Button>

                <Button
                  variant="outline"
                  size="icon"
                  onClick={handleRefreshArtifacts}
                  disabled={isRefreshing || isLoadingArtifacts}
                  title={t("refreshTooltip")}
                >
                  {isRefreshing ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {/* Filter input */}
            <Input
              type="text"
              placeholder={t("filterPlaceholder")}
              value={filterText}
              onChange={(e) => setFilterText(e.target.value)}
              className="w-full"
            />
          </div>

          {/* Artifact list */}
          <div className="flex-1 overflow-auto">
            {isLoadingArtifacts ? (
              <div className="flex items-center justify-center h-full">
                <div className="flex flex-col items-center gap-2">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                  <p className="text-sm text-muted-foreground">
                    {t("loadingArtifacts")}
                  </p>
                </div>
              </div>
            ) : filteredArtifacts.length === 0 ? (
              <div className="flex items-center justify-center h-full">
                <p className="text-sm text-muted-foreground">
                  {artifacts.length === 0
                    ? t("noArtifacts")
                    : t("noMatchingArtifacts")}
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {filteredArtifacts.map((artifact) => {
                  const isSelected = selectedArtifact?.id === artifact.id;
                  return (
                    <div
                      key={artifact.id}
                      className={cn(
                        "group relative rounded-md border p-3 cursor-pointer transition-colors hover:bg-accent",
                        isSelected && "bg-accent border-primary"
                      )}
                      onClick={() => handleArtifactSelect(artifact)}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <p
                            className="text-sm font-medium truncate"
                            title={artifact.id}
                          >
                            {artifact.id}
                          </p>
                          <p className="text-xs text-muted-foreground mt-1">
                            {new Date(artifact.created_at).toLocaleString()}
                          </p>
                        </div>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                          onClick={(e) => handleDeleteClick(artifact, e)}
                        >
                          <Trash2 className="h-3.5 w-3.5 text-destructive" />
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </ResizablePanel>
      <ResizableHandle withHandle />

      {/* File Tree Panel */}
      <ResizablePanel defaultSize={30} minSize={20} maxSize={40}>
        <div className="h-full bg-background p-4">
          <div className="mb-4">
            <h2 className="text-lg font-semibold">{t("filesTitle")}</h2>
          </div>

          {/* Hidden file input */}
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            onChange={handleFileChange}
          />

          <div className="h-[calc(100vh-8rem)]">
            {!selectedArtifact ? (
              <div className="flex items-center justify-center h-full">
                <p className="text-sm text-muted-foreground">
                  {t("selectArtifactPrompt")}
                </p>
              </div>
            ) : isInitialLoading ? (
              <div className="flex items-center justify-center h-full">
                <div className="flex flex-col items-center gap-2">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                  <p className="text-sm text-muted-foreground">
                    {t("loadingFiles")}
                  </p>
                </div>
              </div>
            ) : (
              <div className="h-full flex flex-col">
                {/* Fake root directory with upload button */}
                <div className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-accent hover:text-accent-foreground transition-colors group">
                  <div className="flex items-center gap-1.5">
                    <FolderOpen className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <span className="text-sm">/</span>
                  </div>
                  <button
                    className="shrink-0 p-1 rounded-md bg-primary/10 hover:bg-primary/20 opacity-0 group-hover:opacity-100 transition-all"
                    onClick={() => handleUploadClick("/")}
                    disabled={isUploading}
                    title={t("uploadToRootTooltip")}
                  >
                    {isUploading ? (
                      <Loader2 className="h-3 w-3 animate-spin text-primary" />
                    ) : (
                      <Upload className="h-3 w-3 text-primary" />
                    )}
                  </button>
                </div>

                {/* File tree */}
                <div className="flex-1">
                  <Tree
                    ref={treeRef}
                    data={treeData}
                    openByDefault={false}
                    width="100%"
                    height={750}
                    indent={12}
                    rowHeight={32}
                    onToggle={handleToggle}
                    onSelect={handleSelect}
                  >
                    {(props) => (
                      <Node
                        {...props}
                        loadingNodes={loadingNodes}
                        onUploadClick={handleUploadClick}
                        isUploading={isUploading}
                        t={t}
                      />
                    )}
                  </Tree>
                </div>
              </div>
            )}
          </div>
        </div>
      </ResizablePanel>
      <ResizableHandle withHandle />
      <ResizablePanel>
        <div className="h-full bg-background p-4 overflow-auto">
          <h2 className="mb-4 text-lg font-semibold">{t("contentTitle")}</h2>
          <div className="rounded-md border bg-card p-6">
            {selectedFile && selectedFile.fileInfo ? (
              <div className="space-y-6">
                {/* File header */}
                <div className="border-b pb-4">
                  <h3 className="text-xl font-semibold mb-2">
                    {selectedFile.fileInfo.filename}
                  </h3>
                  <p className="text-sm text-muted-foreground font-mono">
                    {selectedFile.path}
                  </p>
                </div>

                {/* File details */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      {t("mimeType")}
                    </p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {selectedFile.fileInfo.meta.__file_info__.mime}
                    </p>
                  </div>

                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      {t("size")}
                    </p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {selectedFile.fileInfo.meta.__file_info__.size}{" "}
                    </p>
                  </div>

                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      {t("createdAt")}
                    </p>
                    <p className="text-sm bg-muted px-2 py-1 rounded">
                      {new Date(
                        selectedFile.fileInfo.created_at
                      ).toLocaleString()}
                    </p>
                  </div>

                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      {t("updatedAt")}
                    </p>
                    <p className="text-sm bg-muted px-2 py-1 rounded">
                      {new Date(
                        selectedFile.fileInfo.updated_at
                      ).toLocaleString()}
                    </p>
                  </div>
                </div>

                {/* Additional meta information (excluding __file_info__) */}
                {(() => {
                  // eslint-disable-next-line @typescript-eslint/no-unused-vars
                  const { __file_info__, ...additionalMeta } =
                    selectedFile.fileInfo.meta || {};

                  if (Object.keys(additionalMeta).length > 0) {
                    return (
                      <div className="border-t pt-4">
                        <p className="text-sm font-medium text-muted-foreground mb-3">
                          {t("additionalMetadata")}
                        </p>
                        <ReactCodeMirror
                          value={JSON.stringify(additionalMeta, null, 2)}
                          height="200px"
                          theme={resolvedTheme === "dark" ? okaidia : "light"}
                          extensions={[json(), EditorView.lineWrapping]}
                          editable={false}
                          readOnly
                          className="border rounded-md overflow-hidden"
                        />
                      </div>
                    );
                  }
                  return null;
                })()}

                {/* Action buttons - Download and Delete side by side */}
                <div className="border-t pt-4">
                  <div className="flex gap-3">
                    <Button
                      variant="outline"
                      className="flex-1"
                      onClick={handleDownloadClick}
                      disabled={isLoadingDownload}
                    >
                      {isLoadingDownload ? (
                        <>
                          <Loader2 className="h-4 w-4 animate-spin mr-2" />
                          {t("downloading")}
                        </>
                      ) : (
                        <>
                          <Download className="h-4 w-4 mr-2" />
                          {t("download")}
                        </>
                      )}
                    </Button>
                    <Button
                      variant="destructive"
                      className="flex-1"
                      onClick={handleDeleteFileClick}
                      disabled={isDeletingFile}
                    >
                      {isDeletingFile ? (
                        <>
                          <Loader2 className="h-4 w-4 animate-spin mr-2" />
                          {t("deleting")}
                        </>
                      ) : (
                        <>
                          <Trash2 className="h-4 w-4 mr-2" />
                          {t("delete")}
                        </>
                      )}
                    </Button>
                  </div>
                </div>

                {/* Preview section for images */}
                {selectedFile.fileInfo.meta.__file_info__.mime.startsWith(
                  "image/"
                ) && (
                  <div className="border-t pt-6">
                    <p className="text-sm font-medium text-muted-foreground mb-3">
                      {t("preview")}
                    </p>
                    {isLoadingPreview ? (
                      <div className="flex items-center justify-center h-64 bg-muted rounded-md">
                        <div className="flex flex-col items-center gap-2">
                          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                          <p className="text-sm text-muted-foreground">
                            {t("loadingImage")}
                          </p>
                        </div>
                      </div>
                    ) : imageUrl ? (
                      <div className="rounded-md border bg-muted p-4">
                        <div className="relative w-full min-h-[200px]">
                          <Image
                            src={imageUrl}
                            alt={selectedFile.fileInfo.filename}
                            width={800}
                            height={600}
                            className="max-w-full h-auto rounded-md shadow-sm"
                            style={{ objectFit: "contain" }}
                            unoptimized
                          />
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-center justify-center h-64 bg-muted rounded-md">
                        <Button
                          variant="outline"
                          onClick={handlePreviewClick}
                          disabled={isLoadingPreview}
                        >
                          {t("loadPreview")}
                        </Button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                {t("selectFilePrompt")}
              </p>
            )}
          </div>
        </div>
      </ResizablePanel>

      {/* Delete artifact confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteArtifactTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("deleteArtifactDescription")}{" "}
              <span className="font-mono font-semibold">
                {artifactToDelete?.id}
              </span>
              {t("deleteArtifactWarning")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>
              {t("cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteArtifact}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  {t("deleting")}
                </>
              ) : (
                t("delete")
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete file confirmation dialog */}
      <AlertDialog
        open={deleteFileDialogOpen}
        onOpenChange={setDeleteFileDialogOpen}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteFileTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("deleteFileDescription")}{" "}
              <span className="font-mono font-semibold">
                {fileToDelete?.fileInfo?.filename}
              </span>
              {t("deleteFileWarning")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeletingFile}>
              {t("cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteFile}
              disabled={isDeletingFile}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeletingFile ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  {t("deleting")}
                </>
              ) : (
                t("delete")
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Upload file dialog */}
      <AlertDialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
        <AlertDialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <AlertDialogHeader>
            <AlertDialogTitle>{t("uploadFileTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("uploadFileDescription")}
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="space-y-4 py-4">
            {/* File info */}
            <div>
              <label className="text-sm font-medium mb-2 block">
                {t("selectedFile")}
              </label>
              <div className="text-sm bg-muted px-3 py-2 rounded-md font-mono">
                {selectedUploadFile?.name || t("noFileSelected")}
              </div>
            </div>

            {/* Path input */}
            <div>
              <label className="text-sm font-medium mb-2 block">
                {t("uploadPath")}
              </label>
              <Input
                type="text"
                value={uploadPath}
                onChange={(e) => setUploadPath(e.target.value)}
                placeholder={t("uploadPathPlaceholder")}
                className="font-mono"
                disabled={initialUploadPath !== "/"}
              />
              <p className="text-xs text-muted-foreground mt-1">
                {initialUploadPath === "/"
                  ? t("uploadPathHelp")
                  : t("uploadPathHelpLocked")}
              </p>
            </div>

            {/* Meta fields */}
            <div>
              <label className="text-sm font-medium mb-2 block">
                {t("metaInformation")}
              </label>
              <ReactCodeMirror
                value={uploadMetaValue}
                height="200px"
                theme={resolvedTheme === "dark" ? okaidia : "light"}
                extensions={[json(), EditorView.lineWrapping]}
                onChange={handleUploadMetaChange}
                placeholder='{"key": "value"}'
                className="border rounded-md overflow-hidden"
              />
              {uploadMetaError && (
                <p className="mt-2 text-sm text-destructive">{uploadMetaError}</p>
              )}
              <p className="text-xs text-muted-foreground mt-1">
                {t("metaJsonHelp")}
              </p>
            </div>
          </div>

          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={handleUploadCancel}
              disabled={isUploading}
            >
              {t("cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleUploadConfirm}
              disabled={isUploading || !selectedUploadFile || !isUploadMetaValid}
            >
              {isUploading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  {t("uploading")}
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4 mr-2" />
                  {t("upload")}
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </ResizablePanelGroup>
  );
}
