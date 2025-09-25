import { NextRequest, NextResponse } from 'next/server';
import { spawn } from 'child_process';
import path from 'path';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const {
      prompt,
      brief,
      parallel = true,
      provider = 'bedrock',
      model = 'claude-3-5-sonnet',
      outputDir,
      overlays = []
    } = body;

    const description = prompt || brief;
    if (!description) {
      return NextResponse.json(
        { error: 'Prompt or brief is required' },
        { status: 400 }
      );
    }

    console.log('Starting real generation:', {
      description,
      parallel,
      provider,
      model,
      outputDir,
      overlays
    });

    // Path to your QLF binary
    const qlfPath = path.join(process.cwd(), '..', 'bin', 'qlf');

    // Build command arguments
    const args = [
      'generate',
      description,
      '--async',  // Return immediately with WorkflowID
      '--verbose',
      '--provider', provider
    ];

    if (parallel) {
      args.push('--parallel');
    }

    if (model) {
      args.push('--model', model);
    }

    if (outputDir) {
      args.push('--output', outputDir);
    }

    if (overlays && overlays.length > 0) {
      overlays.forEach((overlay: string) => {
        args.push('--overlay', overlay);
      });
    }

    // Execute the real QLF command with --async flag
    return new Promise((resolve, reject) => {
      const child = spawn(qlfPath, args, {
        cwd: path.join(process.cwd(), '..'),
        stdio: ['pipe', 'pipe', 'pipe']
      });

      let stdout = '';
      let stderr = '';

      child.stdout?.on('data', (data) => {
        stdout += data.toString();
      });

      child.stderr?.on('data', (data) => {
        stderr += data.toString();
      });

      child.on('close', (code) => {
        console.log('QLF command output:', stdout);
        console.log('QLF command stderr:', stderr);

        if (code !== 0) {
          resolve(NextResponse.json({
            error: 'QLF command failed',
            details: stderr || 'Unknown error',
            stdout: stdout.substring(0, 500),
            stderr: stderr.substring(0, 500)
          }, { status: 500 }));
          return;
        }

        // Parse the output to extract workflow information
        const workflowMatch = stdout.match(/WorkflowID:\s*([^\s\n]+)/);
        const projectMatch = stdout.match(/ProjectID:\s*([^\s\n]+)/) || stdout.match(/Project:\s*([^\s\n]+)/);

        const workflowId = workflowMatch ? workflowMatch[1] : `factory-${Date.now()}`;
        const projectId = projectMatch ? projectMatch[1] : `project-${Date.now()}`;

        resolve(NextResponse.json({
          workflowId,
          projectId,
          status: 'started',
          mode: parallel ? 'parallel' : 'sequential',
          provider,
          model,
          realBackend: true,
          estimatedDuration: parallel ? '30-60s' : '60-120s',
          stdout: stdout.substring(0, 500),
          stderr: stderr.substring(0, 500)
        }));
      });

      child.on('error', (err) => {
        console.error('QLF spawn error:', err);
        resolve(NextResponse.json({
          error: 'Failed to start QLF command',
          details: err.message,
          realBackend: true
        }, { status: 500 }));
      });
    });

  } catch (error) {
    console.error('Real backend generation error:', error);
    return NextResponse.json(
      {
        error: 'Failed to start generation',
        details: error instanceof Error ? error.message : 'Unknown error',
        realBackend: true
      },
      { status: 500 }
    );
  }
}

export async function GET() {
  return NextResponse.json({
    status: 'healthy',
    service: 'QuantumLayer Platform API (Connected to Real Backend)',
    timestamp: new Date().toISOString(),
    realBackend: true
  });
}