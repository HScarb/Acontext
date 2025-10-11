"use client";

import { useState, useRef, useEffect } from "react";
import Image from "next/image";
import { Tree, NodeRendererProps, TreeApi, NodeApi } from "react-arborist";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
import {
  ChevronRight,
  File,
  Folder,
  FolderOpen,
  Loader2,
  ChevronDown,
  Download,
  Plus,
  Trash2,
  RefreshCw,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  getArtifacts,
  getListFiles,
  getFile,
  createArtifact,
  deleteArtifact,
} from "@/api/models/artifact";
import { Artifact, ListFilesResp, File as FileInfo } from "@/types";

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

function Node({ node, style, dragHandle, loadingNodes }: NodeProps) {
  const indent = node.level * 12;
  const isFolder = node.data.type === "folder";
  const isLoading = loadingNodes.has(node.id);
  const textRef = useRef<HTMLSpanElement>(null);
  const [displayName, setDisplayName] = useState(node.data.name);

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
        "flex items-center cursor-pointer px-2 py-1.5 text-sm rounded-md transition-colors",
        "hover:bg-accent hover:text-accent-foreground",
        node.isSelected && "bg-accent text-accent-foreground",
        node.state.isDragging && "opacity-50"
      )}
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
    </div>
  );
}

export default function ArtifactPage() {
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
  const [isLoadingImage, setIsLoadingImage] = useState(false);
  const [fileUrl, setFileUrl] = useState<string | null>(null);

  // Delete confirmation dialog states
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [artifactToDelete, setArtifactToDelete] = useState<Artifact | null>(
    null
  );
  const [isDeleting, setIsDeleting] = useState(false);

  // Create artifact states
  const [isCreating, setIsCreating] = useState(false);

  // Refresh states
  const [isRefreshing, setIsRefreshing] = useState(false);

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
      id: file.path,
      name: file.filename,
      type: "file",
      path: file.path,
      isLoaded: false,
      fileInfo: file,
    }));
    const directories: TreeNode[] = res.directories.map((directory) => ({
      id: directory,
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
                children: updateNode(files),
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

      // If the deleted artifact is the currently selected one, clear selection
      if (selectedArtifact?.id === artifactToDelete.id) {
        setSelectedArtifact(null);
        setTreeData([]);
        setSelectedFile(null);
      }

      // Reload artifacts list
      await loadArtifacts();
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
      await loadArtifacts();
    } catch (error) {
      console.error("Failed to refresh artifacts:", error);
    } finally {
      setIsRefreshing(false);
    }
  };

  // Load file when a file is selected
  useEffect(() => {
    const loadFile = async () => {
      if (!selectedFile || !selectedArtifact || !selectedFile.fileInfo) {
        setImageUrl(null);
        setFileUrl(null);
        return;
      }

      const mime = selectedFile.fileInfo.meta.__file_info__.mime;
      const isImage = mime.startsWith("image/");

      try {
        setIsLoadingImage(isImage);
        const res = await getFile(
          selectedArtifact.id,
          `${selectedFile.path}${selectedFile.fileInfo.filename}`
        );
        if (res.code !== 0) {
          console.error(res.message);
          setImageUrl(null);
          setFileUrl(null);
          return;
        }

        if (isImage) {
          setImageUrl(res.data?.public_url || null);
          setFileUrl(null);
        } else {
          setFileUrl(res.data?.public_url || null);
          setImageUrl(null);
        }
      } catch (error) {
        console.error("Failed to load file:", error);
        setImageUrl(null);
        setFileUrl(null);
      } finally {
        setIsLoadingImage(false);
      }
    };

    loadFile();
  }, [selectedFile, selectedArtifact]);

  return (
    <ResizablePanelGroup direction="horizontal" className="h-screen">
      <ResizablePanel defaultSize={30} minSize={20} maxSize={40}>
        <div className="h-full bg-background p-4">
          <div className="mb-4 space-y-3">
            <h2 className="text-lg font-semibold">File Explorer</h2>

            {/* Artifact selector with create button */}
            <div className="flex gap-2">
              <DropdownMenu modal={false}>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className="min-w-0 flex-1 justify-between"
                    disabled={isLoadingArtifacts}
                  >
                    {isLoadingArtifacts ? (
                      <>
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span className="ml-2">Loading...</span>
                      </>
                    ) : selectedArtifact ? (
                      <>
                        <span
                          className="mr-2 min-w-0 truncate"
                          title={selectedArtifact.id}
                        >
                          {selectedArtifact.id}
                        </span>
                        <ChevronDown className="h-4 w-4 opacity-50 shrink-0" />
                      </>
                    ) : (
                      <>
                        <span className="text-muted-foreground">
                          Select an artifact
                        </span>
                        <ChevronDown className="h-4 w-4 opacity-50 shrink-0" />
                      </>
                    )}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  className="w-[var(--radix-dropdown-menu-trigger-width)] min-w-[200px]"
                  align="start"
                >
                  {artifacts.map((artifact) => {
                    const isSelected = selectedArtifact?.id === artifact.id;
                    return (
                      <DropdownMenuItem
                        key={artifact.id}
                        title={artifact.id}
                        className="flex items-center justify-between group"
                        disabled={isSelected}
                        onSelect={(e) => e.preventDefault()}
                      >
                        <span
                          className={cn(
                            "truncate flex-1 cursor-pointer",
                            isSelected && "cursor-not-allowed opacity-50"
                          )}
                          onClick={() => !isSelected && handleArtifactSelect(artifact)}
                        >
                          {artifact.id}
                        </span>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity ml-2 shrink-0"
                          onClick={(e) => handleDeleteClick(artifact, e)}
                        >
                          <Trash2 className="h-3.5 w-3.5 text-destructive" />
                        </Button>
                      </DropdownMenuItem>
                    );
                  })}
                </DropdownMenuContent>
              </DropdownMenu>

              <Button
                variant="outline"
                size="icon"
                onClick={handleCreateArtifact}
                disabled={isCreating || isLoadingArtifacts}
                title="Create new artifact"
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
                title="Refresh artifacts"
              >
                {isRefreshing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <div className="h-[calc(100vh-11rem)]">
            {!selectedArtifact ? (
              <div className="flex items-center justify-center h-full">
                <p className="text-sm text-muted-foreground">
                  Select an artifact to view files
                </p>
              </div>
            ) : isInitialLoading ? (
              <div className="flex items-center justify-center h-full">
                <div className="flex flex-col items-center gap-2">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                  <p className="text-sm text-muted-foreground">
                    Loading files...
                  </p>
                </div>
              </div>
            ) : (
              <Tree
                ref={treeRef}
                data={treeData}
                openByDefault={false}
                width="100%"
                height={800}
                indent={12}
                rowHeight={32}
                className="p-2"
                onToggle={handleToggle}
                onSelect={handleSelect}
              >
                {(props) => <Node {...props} loadingNodes={loadingNodes} />}
              </Tree>
            )}
          </div>
        </div>
      </ResizablePanel>
      <ResizableHandle withHandle />
      <ResizablePanel>
        <div className="h-full bg-background p-4 overflow-auto">
          <h2 className="mb-4 text-lg font-semibold">Content</h2>
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
                      MIME Type
                    </p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {selectedFile.fileInfo.meta.__file_info__.mime}
                    </p>
                  </div>

                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      Size
                    </p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {selectedFile.fileInfo.meta.__file_info__.size}{" "}
                    </p>
                  </div>

                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      Created At
                    </p>
                    <p className="text-sm bg-muted px-2 py-1 rounded">
                      {new Date(
                        selectedFile.fileInfo.created_at
                      ).toLocaleString()}
                    </p>
                  </div>

                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">
                      Updated At
                    </p>
                    <p className="text-sm bg-muted px-2 py-1 rounded">
                      {new Date(
                        selectedFile.fileInfo.updated_at
                      ).toLocaleString()}
                    </p>
                  </div>
                </div>

                {/* Image preview */}
                {selectedFile.fileInfo.meta.__file_info__.mime.startsWith(
                  "image/"
                ) ? (
                  <div className="border-t pt-6">
                    <p className="text-sm font-medium text-muted-foreground mb-3">
                      Preview
                    </p>
                    {isLoadingImage ? (
                      <div className="flex items-center justify-center h-64 bg-muted rounded-md">
                        <div className="flex flex-col items-center gap-2">
                          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                          <p className="text-sm text-muted-foreground">
                            Loading image...
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
                        <p className="text-sm text-muted-foreground">
                          Failed to load image
                        </p>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="border-t pt-6">
                    <p className="text-sm font-medium text-muted-foreground mb-3">
                      Download
                    </p>
                    {fileUrl ? (
                      <Button
                        variant="outline"
                        className="w-full"
                        onClick={() => {
                          if (!selectedFile.fileInfo) return;
                          const link = document.createElement("a");
                          link.href = fileUrl;
                          link.download = selectedFile.fileInfo.filename;
                          document.body.appendChild(link);
                          link.click();
                          document.body.removeChild(link);
                        }}
                      >
                        <Download className="h-4 w-4 mr-2" />
                        Click to Download
                      </Button>
                    ) : (
                      <div className="flex items-center justify-center h-20 bg-muted rounded-md">
                        <p className="text-sm text-muted-foreground">
                          No download link available
                        </p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Select a file from the tree to view its content
              </p>
            )}
          </div>
        </div>
      </ResizablePanel>

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Artifact</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete artifact{" "}
              <span className="font-mono font-semibold">
                {artifactToDelete?.id}
              </span>
              ? This action cannot be undone and will delete all files
              associated with this artifact.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteArtifact}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  Deleting...
                </>
              ) : (
                "Delete"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </ResizablePanelGroup>
  );
}
