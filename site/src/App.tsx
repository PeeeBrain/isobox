import React, { useState } from "react";
import {
    Terminal,
    Check,
    Copy,
    Lock,
    ArrowRight,
    ExternalLink,
    Cpu,
    History,
    UserCheck,
    FileCode,
    ShieldAlert,
    AlertTriangle,
    Layers,
} from "lucide-react";
import { Button } from "./components/ui/Button";
import { Badge } from "./components/ui/Badge";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "./components/ui/Card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "./components/ui/Tabs";

const Github = (props: React.SVGProps<SVGSVGElement>) => (
    <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        {...props}
    >
        <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
        <path d="M9 18c-4.51 2-5-2-7-2" />
    </svg>
);

function App() {
    const [copied, setCopied] = useState(false);
    const installCommand =
        "curl -fsSL https://raw.githubusercontent.com/PeeeBrain/isobox/main/install.sh | bash";

    const handleCopy = async () => {
        try {
            await navigator.clipboard.writeText(installCommand);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        } catch (err) {
            console.error("Failed to copy text: ", err);
        }
    };

    const scrollToInstall = (e: React.MouseEvent) => {
        e.preventDefault();
        document
            .getElementById("install")
            ?.scrollIntoView({ behavior: "smooth" });
    };

    const scrollToSecurity = (e: React.MouseEvent) => {
        e.preventDefault();
        document
            .getElementById("security-model")
            ?.scrollIntoView({ behavior: "smooth" });
    };

    return (
        <div className="min-h-screen flex flex-col selection:bg-zinc-800 selection:text-emerald-400">
            {/* Top Navigation */}
            <header className="sticky top-0 z-50 w-full border-b border-zinc-900 bg-zinc-950/80 backdrop-blur-md">
                <div className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <div className="w-5 h-5 bg-zinc-900 border border-zinc-700 flex items-center justify-center rounded-sm">
                            <span className="text-[10px] font-mono font-bold text-emerald-400">
                                i
                            </span>
                        </div>
                        <span className="font-mono text-lg font-bold tracking-tight text-zinc-100">
                            isobox
                        </span>
                    </div>

                    <nav className="hidden md:flex items-center gap-6">
                        <a
                            href="https://github.com/PeeeBrain/isobox"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-sm font-mono text-zinc-400 hover:text-zinc-100 transition-colors flex items-center gap-1"
                        >
                            GitHub <ExternalLink className="w-3 h-3" />
                        </a>
                        <a
                            href="https://github.com/PeeeBrain/isobox#readme"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-sm font-mono text-zinc-400 hover:text-zinc-100 transition-colors"
                        >
                            Docs
                        </a>
                        <a
                            href="#security-model"
                            onClick={scrollToSecurity}
                            className="text-sm font-mono text-zinc-400 hover:text-zinc-100 transition-colors"
                        >
                            Security Model
                        </a>
                        <a
                            href="#install"
                            onClick={scrollToInstall}
                            className="text-sm font-mono text-zinc-400 hover:text-zinc-100 transition-colors"
                        >
                            Install
                        </a>
                    </nav>

                    <div className="flex items-center gap-3">
                        <a
                            href="https://github.com/PeeeBrain/isobox"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-zinc-400 hover:text-zinc-100 md:hidden"
                        >
                            <Github className="w-5 h-5" />
                        </a>
                        <Button
                            variant="outline"
                            size="sm"
                            className="font-mono border-zinc-800 text-zinc-300 hover:bg-zinc-900"
                            onClick={scrollToInstall}
                        >
                            Install
                        </Button>
                    </div>
                </div>
            </header>

            <main className="flex-1 max-w-6xl w-full mx-auto px-4 py-12 md:py-20 space-y-24">
                {/* Hero Section */}
                <section className="text-center space-y-8 flex flex-col items-center">


                    <div className="space-y-4 max-w-3xl">
                        <h1 className="text-4xl md:text-6xl font-bold tracking-tight text-zinc-100 leading-tight md:leading-none">
                            Run coding agents in{" "}
                            <span className="font-mono text-emerald-400">
                                disposable workspaces.
                            </span>
                        </h1>
                        <p className="text-xl md:text-2xl font-mono text-zinc-400 max-w-2xl mx-auto">
                            Promote only reviewed changes.
                        </p>
                    </div>

                    <p className="text-zinc-400 text-base md:text-lg max-w-2xl mx-auto leading-relaxed">
                        isobox provides a workflow and containment boundary for
                        autonomous development. It executes agents inside
                        isolated clone environments, captures detailed action
                        records, and prevents untrusted modifications from
                        reaching your main repository without authorization.
                    </p>

                    <div className="flex gap-4">
                        <Button
                            size="lg"
                            className="font-mono"
                            onClick={scrollToInstall}
                        >
                            Get Started <ArrowRight className="ml-2 w-4 h-4" />
                        </Button>
                        <a
                            href="https://github.com/PeeeBrain/isobox"
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            <Button
                                size="lg"
                                variant="outline"
                                className="font-mono border-zinc-800 hover:bg-zinc-900"
                            >
                                <Github className="mr-2 w-4 h-4" /> View Source
                            </Button>
                        </a>
                    </div>
                </section>

                {/* Terminal Preview */}
                <section className="max-w-4xl mx-auto">
                    <div className="rounded-lg border border-zinc-800 bg-zinc-950 overflow-hidden shadow-2xl terminal-window">
                        {/* Terminal Header */}
                        <div className="flex items-center justify-between px-4 py-3 bg-zinc-900/60 border-b border-zinc-900 terminal-header select-none">
                            <div className="flex gap-1.5">
                                <span className="w-3 h-3 rounded-full bg-zinc-800"></span>
                                <span className="w-3 h-3 rounded-full bg-zinc-800"></span>
                                <span className="w-3 h-3 rounded-full bg-zinc-800"></span>
                            </div>
                            <span className="text-xs font-mono text-zinc-500">
                                isobox@runner:~
                            </span>
                            <div className="w-10"></div>
                        </div>

                        {/* Terminal Body */}
                        <div className="p-5 font-mono text-sm leading-relaxed overflow-x-auto whitespace-pre text-zinc-300">
                            <div className="flex gap-2">
                                <span className="text-emerald-500 select-none">
                                    $
                                </span>
                                <span className="text-zinc-100">
                                    curl -fsSL
                                    https://raw.githubusercontent.com/PeeeBrain/isobox/main/install.sh
                                    | bash
                                </span>
                            </div>
                            <div className="text-zinc-400">
                                Downloading isobox_linux_amd64.tar.gz...
                            </div>
                            <div className="text-zinc-400">
                                Verifying checksum...
                            </div>
                            <div className="text-emerald-400">
                                Installed isobox to /home/user/.local/bin/isobox
                            </div>
                            <div className="mt-4 flex gap-2">
                                <span className="text-emerald-500 select-none">
                                    $
                                </span>
                                <span className="text-zinc-100">
                                    isobox run --source ./repo -- opencode #
                                    Supports other coding agents like claude,
                                    codex, pi and more.
                                </span>
                            </div>
                            <div className="text-emerald-500">
                                ✓ private workspace created
                            </div>
                            <div className="text-emerald-500">
                                ✓ workload completed
                            </div>
                            <div className="text-emerald-500">
                                ✓ stdout, stderr, exit status, policy metadata,
                                and diff captured
                            </div>
                            <div className="text-amber-500">
                                → review task record before promotion
                            </div>
                            <div className="mt-2 flex gap-2">
                                <span className="text-emerald-500 select-none">
                                    $
                                </span>
                                <span className="w-2 h-4 bg-zinc-400 animate-blink"></span>
                            </div>
                        </div>
                    </div>
                </section>

                {/* Install Command Card */}
                <section
                    id="install"
                    className="scroll-mt-24 max-w-3xl mx-auto space-y-6"
                >
                    <div className="text-center space-y-2">
                        <h2 className="text-2xl md:text-3xl font-bold tracking-tight text-zinc-100">
                            Installation
                        </h2>
                        <p className="text-zinc-400 font-mono text-sm">
                            Deploy isobox instantly to your local machine or
                            devcontainer.
                        </p>
                    </div>

                    <Card className="border-zinc-800 bg-zinc-950">
                        <CardHeader className="pb-3 border-b border-zinc-900">
                            <Tabs defaultValue="curl" className="w-full">
                                <div className="flex justify-between items-center flex-wrap gap-4">
                                    <TabsList className="bg-zinc-900 border border-zinc-800">
                                        <TabsTrigger value="curl">
                                            cURL Install
                                        </TabsTrigger>
                                        <TabsTrigger value="manual">
                                            Manual Download
                                        </TabsTrigger>
                                    </TabsList>
                                    <span className="text-xs text-zinc-500 font-mono">
                                        Linux / WSL2 (amd64, arm64)
                                    </span>
                                </div>

                                <TabsContent
                                    value="curl"
                                    className="mt-4 space-y-4"
                                >
                                    <div className="relative flex items-center justify-between bg-zinc-950 border border-zinc-800 rounded px-4 py-3 font-mono text-sm overflow-x-auto text-zinc-100">
                                        <div className="flex items-center gap-2 pr-12 min-w-0">
                                            <span className="text-zinc-500 select-none">
                                                $
                                            </span>
                                            <code className="text-zinc-300 break-all">
                                                {installCommand}
                                            </code>
                                        </div>
                                        <Button
                                            size="icon"
                                            variant="ghost"
                                            onClick={handleCopy}
                                            className="absolute right-2 top-1/2 -translate-y-1/2 h-8 w-8 text-zinc-400 hover:text-zinc-100 border border-zinc-800 bg-zinc-950"
                                            title="Copy to clipboard"
                                        >
                                            {copied ? (
                                                <Check className="w-4 h-4 text-emerald-400" />
                                            ) : (
                                                <Copy className="w-4 h-4" />
                                            )}
                                        </Button>
                                    </div>

                                    <div className="text-xs text-zinc-400 space-y-1.5 font-mono leading-relaxed bg-zinc-900/40 p-3 border border-zinc-900">
                                        <p className="text-zinc-300 font-medium flex items-center gap-1.5">
                                            <Terminal className="w-3.5 h-3.5 text-emerald-400" />{" "}
                                            Installer notes:
                                        </p>
                                        <ul className="list-disc pl-5 space-y-1">
                                            <li>
                                                Installs the latest Linux
                                                amd64/arm64 release binary to{" "}
                                                <code className="text-zinc-200 bg-zinc-900 px-1 font-semibold">
                                                    ~/.local/bin
                                                </code>{" "}
                                                by default.
                                            </li>
                                            <li>
                                                The script detects host
                                                operating system (must be Linux)
                                                and CPU architecture (
                                                <code className="text-zinc-200">
                                                    amd64
                                                </code>{" "}
                                                or{" "}
                                                <code className="text-zinc-200">
                                                    arm64
                                                </code>
                                                ).
                                            </li>
                                            <li>
                                                Automatically pulls matching
                                                GitHub Release assets and
                                                verifies the release
                                                cryptographic checksums.
                                            </li>
                                        </ul>
                                    </div>
                                </TabsContent>

                                <TabsContent
                                    value="manual"
                                    className="mt-4 space-y-4"
                                >
                                    <div className="text-sm font-mono text-zinc-400 space-y-3 leading-relaxed">
                                        <p>
                                            If you prefer not to pipe curl
                                            output to bash, you can download the
                                            release directly:
                                        </p>
                                        <ol className="list-decimal pl-5 space-y-2 text-zinc-300">
                                            <li>
                                                Navigate to the official{" "}
                                                <a
                                                    href="https://github.com/PeeeBrain/isobox/releases"
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="text-emerald-400 hover:underline inline-flex items-center gap-0.5"
                                                >
                                                    GitHub Releases{" "}
                                                    <ExternalLink className="w-3 h-3" />
                                                </a>{" "}
                                                page.
                                            </li>
                                            <li>
                                                Download the latest{" "}
                                                <code className="text-zinc-100 bg-zinc-900 px-1">
                                                    tar.gz
                                                </code>{" "}
                                                archive matching your
                                                architecture (e.g.,{" "}
                                                <code className="text-zinc-100">
                                                    isobox_linux_amd64.tar.gz
                                                </code>
                                                ).
                                            </li>
                                            <li>
                                                Extract the archive:
                                                <pre className="mt-1.5 bg-zinc-950 border border-zinc-800 p-2 text-zinc-400 text-xs rounded">
                                                    tar -xzf
                                                    isobox_linux_amd64.tar.gz
                                                </pre>
                                            </li>
                                            <li>
                                                Move the extracted{" "}
                                                <code className="text-zinc-100">
                                                    isobox
                                                </code>{" "}
                                                binary to your executable path
                                                (e.g.,{" "}
                                                <code className="text-zinc-100">
                                                    ~/.local/bin/
                                                </code>{" "}
                                                or{" "}
                                                <code className="text-zinc-100">
                                                    /usr/local/bin/
                                                </code>
                                                ).
                                            </li>
                                        </ol>
                                    </div>
                                </TabsContent>
                            </Tabs>
                        </CardHeader>
                    </Card>
                </section>

                {/* Feature Cards Section */}
                <section className="space-y-8">
                    <div className="text-center space-y-2">
                        <h2 className="text-2xl md:text-3xl font-bold tracking-tight text-zinc-100">
                            Core Capabilities
                        </h2>
                        <p className="text-zinc-400 font-mono text-sm">
                            Fine-grained boundaries built specifically for
                            coding-agent autonomy.
                        </p>
                    </div>

                    <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
                        <Card className="terminal-card border-zinc-800 bg-zinc-950/40">
                            <CardHeader className="pb-2">
                                <Layers className="w-8 h-8 text-zinc-400 mb-2" />
                                <CardTitle className="text-zinc-100">
                                    Disposable Workspaces
                                </CardTitle>
                                <CardDescription>
                                    CONTAINMENT BOUNDARY
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm text-zinc-400 leading-relaxed">
                                isobox duplicates the target codebase into a
                                clean, isolated directory structure for
                                execution. Agents interact solely with this
                                sandbox clone, preventing stray scripts from
                                modifying your active git state.
                            </CardContent>
                        </Card>

                        <Card className="terminal-card border-zinc-800 bg-zinc-950/40">
                            <CardHeader className="pb-2">
                                <History className="w-8 h-8 text-zinc-400 mb-2" />
                                <CardTitle className="text-zinc-100">
                                    Durable Task Records
                                </CardTitle>
                                <CardDescription>
                                    COMPLETE METADATA LOGGING
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm text-zinc-400 leading-relaxed">
                                Captures standard output (stdout), error output
                                (stderr), command exit status, execution
                                execution time, policy metadata, and final git
                                diffs, package into a portable, immutable task
                                record file.
                            </CardContent>
                        </Card>

                        <Card className="terminal-card border-zinc-800 bg-zinc-950/40">
                            <CardHeader className="pb-2">
                                <UserCheck className="w-8 h-8 text-zinc-400 mb-2" />
                                <CardTitle className="text-zinc-100">
                                    Review-Gated Promotion
                                </CardTitle>
                                <CardDescription>
                                    HUMAN-IN-THE-LOOP CONTROL
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm text-zinc-400 leading-relaxed">
                                Agent-modified files reside strictly within the
                                disposable clone. Changes are never merged back
                                to your trusted workspace until a human
                                developer explicitly approves the captured diff
                                records.
                            </CardContent>
                        </Card>

                        <Card className="terminal-card border-zinc-800 bg-zinc-950/40">
                            <CardHeader className="pb-2">
                                <Lock className="w-8 h-8 text-zinc-400 mb-2" />
                                <CardTitle className="text-zinc-100">
                                    Policy Enforcement
                                </CardTitle>
                                <CardDescription>
                                    TRANSPARENT CONTROL
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm text-zinc-400 leading-relaxed">
                                Enforces fine-grained operational policies for
                                workspace executions. It reports exact execution
                                statuses, errors, and policy violations directly
                                without hiding safety boundaries.
                            </CardContent>
                        </Card>

                        <Card className="terminal-card border-zinc-800 bg-zinc-950/40">
                            <CardHeader className="pb-2">
                                <Cpu className="w-8 h-8 text-zinc-400 mb-2" />
                                <CardTitle className="text-zinc-100">
                                    Agent-Independent Workflow
                                </CardTitle>
                                <CardDescription>TOOL AGNOSTIC</CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm text-zinc-400 leading-relaxed">
                                Works wrapping any CLI-based tool, agent system,
                                or command block. You don't need to rewrite your
                                agent's LLM prompts or modify its code—isobox
                                operates at the workflow command level.
                            </CardContent>
                        </Card>

                        <Card className="terminal-card border-zinc-800 bg-zinc-950/40">
                            <CardHeader className="pb-2">
                                <FileCode className="w-8 h-8 text-zinc-400 mb-2" />
                                <CardTitle className="text-zinc-100">
                                    Linux / WSL2 First
                                </CardTitle>
                                <CardDescription>
                                    DEVELOPER COMPATIBILITY
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm text-zinc-400 leading-relaxed">
                                Engineered for Linux-based workflows and Windows
                                Subsystem for Linux (WSL2). Provides close shell
                                integration, native filesystem performance, and
                                low-latency system wrapper execution.
                            </CardContent>
                        </Card>
                    </div>
                </section>

                {/* Agent Skill Integration Section */}
                <section
                    id="agent-skill"
                    className="scroll-mt-24 max-w-4xl mx-auto space-y-6"
                >
                    <div className="text-center space-y-2">
                        <h2 className="text-2xl md:text-3xl font-bold tracking-tight text-zinc-100">
                            Cooperative Agent Skill
                        </h2>
                        <p className="text-zinc-400 font-mono text-sm">
                            We ship{" "}
                            <code className="text-emerald-400 bg-zinc-900 px-1 py-0.5">
                                isobox-agent-guide
                            </code>{" "}
                            to safely run autonomous agents in your codebase.
                        </p>
                    </div>

                    <div className="grid md:grid-cols-2 gap-6">
                        <Card className="border-zinc-800 bg-zinc-950/40 p-6 space-y-4">
                            <div className="flex items-center gap-2 text-zinc-100">
                                <div className="p-2 bg-emerald-950/40 border border-emerald-900/50 rounded text-emerald-400">
                                    <Cpu className="w-5 h-5" />
                                </div>
                                <div>
                                    <h3 className="font-semibold text-lg">
                                        isobox-agent-guide
                                    </h3>
                                    <p className="text-xs text-zinc-500 font-mono">
                                        AGENT RUNTIME CONVENTION
                                    </p>
                                </div>
                            </div>

                            <p className="text-sm text-zinc-400 leading-relaxed">
                                The companion skill operates agents in{" "}
                                <strong>Cooperative Safe Mode</strong>. It
                                defines system instructions and enforcement
                                rules that direct agents to use isobox sandboxes
                                natively.
                            </p>

                            <ul className="space-y-3 text-sm text-zinc-300 font-mono">
                                <li className="flex items-start gap-2">
                                    <span className="text-emerald-500 mt-0.5 font-bold">
                                        ✓
                                    </span>
                                    <span>
                                        <strong>Default Command Routing</strong>
                                        : Directs the agent to wrap all terminal
                                        commands inside{" "}
                                        <code>
                                            isobox tool -- &lt;command&gt;
                                        </code>
                                        .
                                    </span>
                                </li>
                                <li className="flex items-start gap-2">
                                    <span className="text-emerald-500 mt-0.5 font-bold">
                                        ✓
                                    </span>
                                    <span>
                                        <strong>Durable Task Logging</strong>:
                                        Automatically captures exit status,
                                        stdout, stderr, and git diffs for
                                        execution review.
                                    </span>
                                </li>
                                <li className="flex items-start gap-2">
                                    <span className="text-emerald-500 mt-0.5 font-bold">
                                        ✓
                                    </span>
                                    <span>
                                        <strong>Promotion Gate</strong>:
                                        Enforces that proposed code changes are
                                        never merged back to the repository
                                        without explicit developer approval.
                                    </span>
                                </li>
                            </ul>
                        </Card>

                        <Card className="border-zinc-800 bg-zinc-950/40 p-6 flex flex-col justify-between space-y-4">
                            <div className="space-y-3">
                                <h3 className="font-semibold text-lg text-zinc-100 font-mono">
                                    Install the Skill
                                </h3>
                                <p className="text-sm text-zinc-400 leading-relaxed">
                                    Install the skill in your project or global
                                    configuration using the agent skills package
                                    manager:
                                </p>

                                <div className="space-y-4 pt-2">
                                    <div className="space-y-1.5">
                                        <span className="text-xs text-zinc-500 font-mono block">
                                            PROJECT INSTALL (RECOMMENDED)
                                        </span>
                                        <div className="relative bg-zinc-950 border border-zinc-900 rounded p-2 text-xs font-mono text-zinc-300 overflow-x-auto">
                                            <code>
                                                npx skills add
                                                PeeeBrain/isobox@isobox-agent-guide
                                            </code>
                                        </div>
                                    </div>

                                    <div className="space-y-1.5">
                                        <span className="text-xs text-zinc-500 font-mono block">
                                            GLOBAL INSTALL
                                        </span>
                                        <div className="relative bg-zinc-950 border border-zinc-900 rounded p-2 text-xs font-mono text-zinc-300 overflow-x-auto">
                                            <code>
                                                npx skills add
                                                PeeeBrain/isobox@isobox-agent-guide
                                                -g
                                            </code>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <div className="text-xs text-zinc-500 font-mono border-t border-zinc-900 pt-3">
                                Once added, your agent assistant automatically
                                discovers the instructions and begins wrapping
                                its execution shell.
                            </div>
                        </Card>
                    </div>
                </section>

                {/* Security Model Section */}
                <section
                    id="security-model"
                    className="scroll-mt-24 max-w-4xl mx-auto"
                >
                    <div className="relative border border-amber-900/60 bg-amber-950/5 p-6 md:p-8 rounded-lg overflow-hidden space-y-6 terminal-glow-amber">
                        <div className="absolute top-0 right-0 w-24 h-24 bg-amber-500/5 blur-xl rounded-full"></div>

                        <div className="flex flex-col md:flex-row gap-4 items-start">
                            <div className="p-3 bg-amber-950/40 border border-amber-900/50 rounded text-amber-400">
                                <ShieldAlert className="w-6 h-6" />
                            </div>
                            <div className="space-y-2">
                                <div className="flex items-center gap-2">
                                    <Badge variant="warning">
                                        Security Model
                                    </Badge>
                                </div>
                                <h3 className="text-xl md:text-2xl font-semibold tracking-tight text-zinc-100">
                                    Containment & Policy Boundaries, Not Magic
                                    Sandboxing
                                </h3>
                            </div>
                        </div>

                        <div className="space-y-4 text-zinc-400 text-sm md:text-base leading-relaxed">
                            <p>
                                A security system is only as useful as it is
                                transparent.
                                <strong className="text-zinc-300">
                                    {" "}
                                    isobox does not claim perfect isolation.
                                </strong>{" "}
                                It is not a fully isolated hypervisor, nor does
                                it run coding agents inside cryptographically
                                secured hardware enclaves by default.
                            </p>

                            <div className="grid md:grid-cols-2 gap-6 pt-2">
                                <div className="space-y-2 border-l-2 border-zinc-800 pl-4">
                                    <h4 className="text-zinc-200 font-semibold font-mono text-sm flex items-center gap-1.5">
                                        <AlertTriangle className="w-3.5 h-3.5 text-amber-500" />{" "}
                                        What it is NOT:
                                    </h4>
                                    <ul className="list-disc pl-4 space-y-1 text-xs md:text-sm">
                                        <li>
                                            Not a kernel-level virtualization
                                            boundary.
                                        </li>
                                        <li>
                                            Not a protection layer against
                                            low-level local privilege escalation
                                            or hardware attacks.
                                        </li>
                                        <li>
                                            Not a sandbox that blocks all
                                            system-call operations.
                                        </li>
                                    </ul>
                                </div>

                                <div className="space-y-2 border-l-2 border-zinc-800 pl-4">
                                    <h4 className="text-zinc-200 font-semibold font-mono text-sm flex items-center gap-1.5">
                                        <Check className="w-3.5 h-3.5 text-emerald-500" />{" "}
                                        What it is:
                                    </h4>
                                    <ul className="list-disc pl-4 space-y-1 text-xs md:text-sm">
                                        <li>
                                            A workspace directory clone shield
                                            to protect your active code branch.
                                        </li>
                                        <li>
                                            An automation capture system that
                                            records every output, error, and
                                            diff.
                                        </li>
                                        <li>
                                            A manual verification gate ensuring
                                            changes require human confirmation.
                                        </li>
                                    </ul>
                                </div>
                            </div>

                            <p className="pt-2 font-mono text-xs text-zinc-500 border-t border-zinc-900">
                                Current design focus is optimized for Linux /
                                WSL2-first developer tooling configurations. By
                                separating execution from source state, isobox
                                limits damage from runaway or mistaken agent
                                executions.
                            </p>
                        </div>
                    </div>
                </section>
            </main>

            {/* Footer */}
            <footer className="border-t border-zinc-900 bg-zinc-950 mt-20">
                <div className="max-w-6xl mx-auto px-4 py-12 flex flex-col md:flex-row items-center justify-between gap-6">
                    <div className="flex flex-col items-center md:items-start gap-2">
                        <div className="flex items-center gap-2">
                            <div className="w-4 h-4 bg-zinc-900 border border-zinc-700 flex items-center justify-center rounded-sm">
                                <span className="text-[8px] font-mono font-bold text-emerald-400">
                                    i
                                </span>
                            </div>
                            <span className="font-mono text-sm font-bold tracking-tight text-zinc-100">
                                isobox
                            </span>
                        </div>
                        <p className="text-xs text-zinc-500 font-mono">
                            Secure workspaces for coding agents.
                        </p>
                    </div>

                    <div className="flex flex-wrap justify-center gap-x-8 gap-y-2">
                        <a
                            href="https://github.com/PeeeBrain/isobox"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs font-mono text-zinc-500 hover:text-zinc-300"
                        >
                            GitHub
                        </a>
                        <a
                            href="https://github.com/PeeeBrain/isobox#readme"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs font-mono text-zinc-500 hover:text-zinc-300"
                        >
                            Documentation
                        </a>
                        <a
                            href="https://github.com/PeeeBrain/isobox/releases"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs font-mono text-zinc-500 hover:text-zinc-300"
                        >
                            Releases
                        </a>
                        <a
                            href="#install"
                            onClick={scrollToInstall}
                            className="text-xs font-mono text-zinc-500 hover:text-zinc-300"
                        >
                            Install
                        </a>
                    </div>

                    {/* Footer right spacing */}
                    <div className="text-center md:text-right space-y-1"></div>
                </div>
            </footer>
        </div>
    );
}

export default App;
