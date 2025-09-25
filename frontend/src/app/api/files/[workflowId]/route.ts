import { NextRequest, NextResponse } from 'next/server';
import { exec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import fs from 'fs';

const execAsync = promisify(exec);

export async function GET(
  request: NextRequest,
  { params }: { params: { workflowId: string } }
) {
  try {
    const workflowId = params.workflowId;

    if (!workflowId) {
      return NextResponse.json(
        { error: 'Workflow ID is required' },
        { status: 400 }
      );
    }

    // Find the project directory for this workflow
    try {
      const generatedDir = path.join(process.cwd(), '..', 'generated');
      const { stdout: lsOutput } = await execAsync(`ls -1t ${generatedDir} | grep project- | head -1`);
      const projectId = lsOutput.trim();

      if (!projectId) {
        return NextResponse.json({ error: 'No project directory found' }, { status: 404 });
      }

      const outputDir = path.join(generatedDir, projectId);

      // Check if directory exists
      const { stdout: dirExists } = await execAsync(`test -d ${outputDir} && echo exists || echo missing`);
      if (!dirExists.includes('exists')) {
        return NextResponse.json({ error: 'Project directory not found' }, { status: 404 });
      }

      // Recursively list all files in the output directory
      const files = await getFileList(outputDir);

      return NextResponse.json(files);

    } catch (cmdError) {
      console.error('Failed to get files:', cmdError);
      return NextResponse.json({ error: 'Failed to list files' }, { status: 500 });
    }

  } catch (error) {
    console.error('Files API error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

async function getFileList(dir: string, basePath: string = ''): Promise<string[]> {
  const files: string[] = [];

  try {
    const entries = await fs.promises.readdir(dir, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);
      const relativePath = basePath ? path.join(basePath, entry.name) : entry.name;

      if (entry.isDirectory()) {
        // Skip common directories that aren't useful to show
        if (!['node_modules', '.git', '__pycache__', '.venv'].includes(entry.name)) {
          const subFiles = await getFileList(fullPath, relativePath);
          files.push(...subFiles);
        }
      } else {
        // Only include actual code/config files
        const ext = path.extname(entry.name).toLowerCase();
        const validExtensions = ['.py', '.go', '.js', '.ts', '.tsx', '.jsx', '.json', '.yml', '.yaml', '.sql', '.md', '.txt', '.env', '.dockerfile'];

        if (validExtensions.includes(ext) || entry.name.startsWith('.env') || entry.name === 'Dockerfile' || entry.name === 'Makefile') {
          files.push(relativePath);
        }
      }
    }
  } catch (error) {
    console.error(`Error reading directory ${dir}:`, error);
  }

  return files;
}