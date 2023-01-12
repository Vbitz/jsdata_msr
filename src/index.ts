import * as http from 'http';
import ts from 'typescript';

function getBody(req: http.IncomingMessage): Promise<string> {
  return new Promise((res, rej) => {
    const body: Uint8Array[] = [];
    req
      .on('data', chunk => {
        body.push(chunk);
      })
      .on('end', () => {
        const bodyString = Buffer.concat(body).toString();
        if (body) {
          res(bodyString);
        }
      })
      .on('error', err => {
        rej(err);
      });
  });
}

interface Request {
  filename: string;
  fileContents: string;
}

const CURRENT_VERSION = 2;

interface Response {
  version: number;
  processTime: number;
  features: Record<string, boolean>;
}

function gatherFeatures(sourceFile: ts.SourceFile): Record<string, boolean> {
  const ret: Record<string, boolean> = {};

  const walkNode = (node: ts.Node) => {
    if (ts.isSatisfiesExpression(node)) {
      ret['SatisfiesExpression'] = true;
    } else if (ts.isPropertyDeclaration(node)) {
      for (const mod of node.modifiers || []) {
        if (mod.kind === ts.SyntaxKind.AccessorKeyword) {
          ret['AccessorKeyword'] = true;
        }
      }
    } else if (ts.isInferTypeNode(node)) {
      if (node.typeParameter.constraint !== undefined) {
        ret['ExtendsConstraintOnInfer'] = true;
      }
    } else if (ts.isTypeParameterDeclaration(node)) {
      for (const mod of node.modifiers || []) {
        if (
          mod.kind === ts.SyntaxKind.OutKeyword ||
          mod.kind === ts.SyntaxKind.InKeyword
        ) {
          ret['VarianceAnnotationsOnTypeParameter'] = true;
        }
      }
    } else if (ts.isImportSpecifier(node)) {
      if (node.isTypeOnly) {
        ret['TypeModifierOnImportName'] = true;
      }
    } else if (ts.isImportDeclaration(node)) {
      if (node.assertClause !== undefined) {
        ret['ImportAssertion'] = true;
      }
    } else if (ts.isClassStaticBlockDeclaration(node)) {
      ret['StaticBlockInClass'] = true;
    } else if (ts.isMethodDeclaration(node)) {
      for (const mod of node.modifiers || []) {
        if (mod.kind === ts.SyntaxKind.OverrideKeyword) {
          ret['OverrideOnClassMethod'] = true;
        }
      }
    } else if (ts.isConstructorTypeNode(node)) {
      for (const mod of node.modifiers || []) {
        if (mod.kind === ts.SyntaxKind.AbstractKeyword) {
          ret['AbstractConstructSignature'] = true;
        }
      }
    } else if (ts.isTemplateLiteralTypeNode(node)) {
      ret['TemplateLiteralType'] = true;
    } else if (ts.isMappedTypeNode(node) && node.nameType !== undefined) {
      ret['RemappedNameInMappedType'] = true;
    } else if (ts.isNamedTupleMember(node)) {
      ret['NamedTupleMember'] = true;
    } else if (ts.isBinaryExpression(node)) {
      if (
        node.operatorToken.kind === ts.SyntaxKind.QuestionQuestionEqualsToken ||
        node.operatorToken.kind === ts.SyntaxKind.BarBarEqualsToken ||
        node.operatorToken.kind === ts.SyntaxKind.AmpersandAmpersandEqualsToken
      ) {
        ret['ShortCircuitAssignment'] = true;
      }
    }

    ts.forEachChild(node, walkNode);
  };

  walkNode(sourceFile);

  return ret;
}

async function processRequest(body: string): Promise<string> {
  const request: Request = JSON.parse(body);

  const start = process.hrtime.bigint();

  const sourceFile = ts.createSourceFile(
    request.filename,
    request.fileContents,
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TS
  );

  const features = gatherFeatures(sourceFile);

  const end = process.hrtime.bigint();

  const response: Response = {
    version: CURRENT_VERSION,
    processTime: Number(end - start),
    features,
  };

  return JSON.stringify(response);
}

function asyncWrap(
  func: (req: http.IncomingMessage, res: http.ServerResponse) => Promise<void>
): http.RequestListener<
  typeof http.IncomingMessage,
  typeof http.ServerResponse
> {
  return (req, res) => {
    func(req, res).catch(err => {
      res.writeHead(500, 'Internal Server Error', {
        'Content-Type': 'application/json',
      });
      res.end(JSON.stringify({Error: err.toString()}));
    });
  };
}

const server = http.createServer(
  asyncWrap(async (req, res) => {
    const body = await getBody(req);
    const response = await processRequest(body);
    res.writeHead(200, undefined, {
      'Content-Type': 'application/json',
    });
    res.end(response);
  })
);

console.log('Listening on http://localhost:5123/');

server.listen(5123);
