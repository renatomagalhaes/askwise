# Runbook de Suporte — TechSupport Ltda.

## 1. Erro 5032: Timeout de Conexão com API

### Descrição
O erro 5032 ocorre quando o serviço CloudSync não consegue estabelecer conexão com a API externa dentro do tempo limite configurado (padrão: 30 segundos).

### Causas Comuns
1. **Rede instável**: Latência acima de 500ms entre o servidor e a API externa
2. **API indisponível**: O provedor externo pode estar em manutenção
3. **Firewall bloqueando**: Regras de firewall corporativo podem estar bloqueando a porta 443
4. **Certificado expirado**: O certificado TLS do provedor pode ter expirado

### Resolução Passo a Passo
1. Verificar conectividade: `ping api.cloudprovider.com`
2. Testar porta: `telnet api.cloudprovider.com 443`
3. Verificar certificado: `openssl s_client -connect api.cloudprovider.com:443`
4. Checar status do provedor: https://status.cloudprovider.com
5. Se o problema persistir, aumentar o timeout em `config/cloudsync.yaml`:
   ```yaml
   cloudsync:
     timeout_seconds: 60
     retry_count: 3
   ```
6. Reiniciar o serviço: `systemctl restart cloudsync`

### Escalação
Se nenhum dos passos resolver, escalar para o Time de Infraestrutura (Slack: #infra-oncall).

---

## 2. Erro 4010: Falha de Autenticação OAuth

### Descrição
O erro 4010 indica que o token OAuth expirou ou foi revogado. Isso afeta todos os endpoints que requerem autenticação no sistema DataSync.

### Causas Comuns
1. **Token expirado**: Tokens têm validade de 24 horas
2. **Credenciais rotacionadas**: O client secret pode ter sido alterado no portal do provedor
3. **Clock skew**: Diferença de horário entre servidor e provedor de identidade

### Resolução Passo a Passo
1. Verificar validade do token: `curl -H "Authorization: Bearer $TOKEN" https://auth.provider.com/verify`
2. Regenerar token: `datasync auth refresh --force`
3. Se o client secret mudou, atualizar em `config/secrets.yaml`
4. Verificar sincronização de horário: `ntpq -p`
5. Forçar renovação completa: `datasync auth reset && datasync auth login`

### Escalação
Se o erro persistir após renovação, contatar o Time de Segurança (Slack: #sec-team).

---

## 3. Erro 6001: Disco Cheio no Servidor de Logs

### Descrição
O erro 6001 é emitido quando a partição de logs (/var/log) atinge 95% de utilização. O sistema para de gravar logs e pode apresentar comportamento degradado.

### Causas Comuns
1. **Logs não rotacionados**: logrotate pode ter parado
2. **Log level em DEBUG**: Gera volume excessivo em produção
3. **Ataque ou abuso**: Requisições maliciosas gerando muitos erros

### Resolução Passo a Passo
1. Verificar uso do disco: `df -h /var/log`
2. Identificar arquivos grandes: `du -sh /var/log/* | sort -rh | head -20`
3. Limpar logs antigos: `find /var/log -name "*.gz" -mtime +7 -delete`
4. Verificar logrotate: `logrotate -d /etc/logrotate.conf`
5. Se logrotate estiver quebrado: `logrotate -f /etc/logrotate.d/app`
6. Alterar log level para INFO em produção:
   ```yaml
   logging:
     level: INFO
     max_size_mb: 100
     max_backups: 5
   ```

### Prevenção
- Monitorar uso de disco com alertas em 80% e 90%
- Nunca usar DEBUG em produção por mais de 1 hora
- Implementar log sampling para endpoints de alto tráfego

---

## 4. Procedimento: Deploy em Produção

### Pré-requisitos
- Branch `main` com todos os testes passando (CI verde)
- Aprovação de pelo menos 2 revisores no PR
- Comunicação no canal #deploy com janela de deploy

### Passos
1. **Preparação**:
   - Verificar CI: `gh pr checks`
   - Fazer backup do banco: `./scripts/backup-db.sh`
   - Ativar modo de manutenção: `kubectl scale deployment app --replicas=0`

2. **Deploy**:
   - Executar deploy: `make deploy ENV=production`
   - Aguardar rollout: `kubectl rollout status deployment/app --timeout=5m`
   - Verificar health: `curl https://app.techsupport.com/health`

3. **Validação**:
   - Executar smoke tests: `make test-smoke ENV=production`
   - Verificar logs: `kubectl logs -f deployment/app --since=5m`
   - Confirmar métricas no Grafana: dashboard "Production Overview"

4. **Rollback** (se necessário):
   - `kubectl rollout undo deployment/app`
   - Comunicar no #deploy que houve rollback
   - Abrir post-mortem se necessário

---

## 5. FAQ — Perguntas Frequentes

### Como resetar a senha de um usuário?
Execute: `datasync admin reset-password --user email@empresa.com`
O usuário receberá um link de reset por email (válido por 1 hora).

### Como adicionar um novo integration partner?
1. Cadastrar no portal admin: https://admin.techsupport.com/partners
2. Gerar API key: Painel > Integrações > Nova Chave
3. Compartilhar chave via canal seguro (nunca por email)
4. Documentar na wiki: https://wiki.techsupport.com/integracoes

### Qual o SLA de resposta para incidentes?
| Severidade | Tempo de Resposta | Tempo de Resolução |
|------------|-------------------|--------------------|
| P1 (Crítico) | 15 minutos | 4 horas |
| P2 (Alto) | 1 hora | 8 horas |
| P3 (Médio) | 4 horas | 24 horas |
| P4 (Baixo) | 1 dia útil | 5 dias úteis |

### Qual a política de retenção de dados?
- Logs de aplicação: 30 dias
- Logs de auditoria: 1 ano
- Backups de banco: 90 dias
- Dados de usuário: 5 anos após último acesso (LGPD)
